// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import (
	"sort"
	"strings"
)

// Apply runs a Query over records and returns the resulting View. It is a pure
// function of its inputs — no clock, no map-order dependence, no mutation of
// records — which is exactly why a MemoryProxy and a remote grpcproxy can share
// it and produce byte-identical Views: the client and the server call the same
// Apply on the same rows with the same Query.
//
// The pipeline is: filter, then compute the grand aggregates over the filtered
// set, then sort, then either group (and paginate the groups) or paginate the
// rows. Sorting before grouping keeps each group's rows in the query's order and
// makes the group order itself deterministic.
func Apply(records []Record, q Query) View {
	filtered := filter(records, q.Filters)
	view := View{
		Total:      len(filtered),
		Aggregates: aggregate(filtered, q.Aggs),
	}
	sortRows(filtered, q.Sorts)

	if q.GroupBy == "" {
		lo, hi := pageBounds(len(filtered), q.Offset, q.Limit)
		view.Rows = cloneRows(filtered[lo:hi])
		return view
	}

	groups := group(filtered, q.GroupBy, q.Aggs)
	lo, hi := pageBounds(len(groups), q.Offset, q.Limit)
	view.Groups = groups[lo:hi]
	return view
}

// filter keeps the rows satisfying every predicate (ANDed).
func filter(records []Record, filters []Filter) []Record {
	out := make([]Record, 0, len(records))
	for _, r := range records {
		if matchesAll(r, filters) {
			out = append(out, r)
		}
	}
	return out
}

// matchesAll reports whether r satisfies all filters.
func matchesAll(r Record, filters []Filter) bool {
	for _, f := range filters {
		if !match(r, f) {
			return false
		}
	}
	return true
}

// match evaluates one predicate against a row. A missing cell never matches.
func match(r Record, f Filter) bool {
	cell, ok := r[f.Field]
	if !ok {
		return false
	}
	switch f.Op {
	case OpEq:
		return cell.compare(f.Value) == 0
	case OpNe:
		return cell.compare(f.Value) != 0
	case OpLt:
		return cell.compare(f.Value) < 0
	case OpLe:
		return cell.compare(f.Value) <= 0
	case OpGt:
		return cell.compare(f.Value) > 0
	case OpGe:
		return cell.compare(f.Value) >= 0
	default: // OpContains
		return cell.Kind == KindString && f.Value.Kind == KindString &&
			strings.Contains(cell.Str, f.Value.Str)
	}
}

// sortRows stably orders rows by the sort keys in place (primary key first). A
// row lacking a key's field compares as the zero Value of no particular kind, so
// missing cells cluster consistently. With no keys it is a no-op.
func sortRows(rows []Record, keys []Sort) {
	if len(keys) == 0 {
		return
	}
	sort.SliceStable(rows, func(i, j int) bool {
		for _, k := range keys {
			c := rows[i][k.Field].compare(rows[j][k.Field])
			if c == 0 {
				continue
			}
			if k.Desc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
}

// group buckets rows by the value of field, preserving first-appearance order of
// the keys (rows are pre-sorted, so this is the query's order), and computes each
// group's aggregates.
func group(rows []Record, field string, aggs []Agg) []Group {
	order := make([]Value, 0)
	buckets := make(map[Value][]Record)
	for _, r := range rows {
		key := r[field]
		if _, seen := buckets[key]; !seen {
			order = append(order, key)
		}
		buckets[key] = append(buckets[key], r)
	}
	out := make([]Group, 0, len(order))
	for _, key := range order {
		members := buckets[key]
		out = append(out, Group{
			Key:        key,
			Rows:       cloneRows(members),
			Aggregates: aggregate(members, aggs),
		})
	}
	return out
}

// aggregate computes every requested aggregate over rows. It returns nil when no
// aggregates are requested, so an aggregate-free View has a nil (not empty) map —
// a single canonical shape for "no aggregates".
func aggregate(rows []Record, aggs []Agg) map[string]Value {
	if len(aggs) == 0 {
		return nil
	}
	out := make(map[string]Value, len(aggs))
	for _, a := range aggs {
		out[a.label()] = reduce(rows, a)
	}
	return out
}

// reduce computes one aggregate. Count is an integer; sum/avg are floats; min/max
// carry the extreme value with its own kind. An empty input yields a zero-count,
// a zero sum/avg, and a zero-Value min/max.
func reduce(rows []Record, a Agg) Value {
	if a.Func == AggCount {
		return Int(int64(len(rows)))
	}
	var sum float64
	var n int
	var ext Value
	var have bool
	for _, r := range rows {
		cell, ok := r[a.Field]
		if !ok {
			continue
		}
		n++
		sum += cell.num()
		if !have || (a.Func == AggMin && cell.compare(ext) < 0) ||
			(a.Func == AggMax && cell.compare(ext) > 0) {
			ext, have = cell, true
		}
	}
	switch a.Func {
	case AggSum:
		return Float(sum)
	case AggAvg:
		if n == 0 {
			return Float(0)
		}
		return Float(sum / float64(n))
	default: // AggMin, AggMax
		return ext
	}
}

// cloneRows deep-copies a slice of rows so a View never aliases a proxy's storage.
func cloneRows(rows []Record) []Record {
	out := make([]Record, len(rows))
	for i, r := range rows {
		out[i] = r.clone()
	}
	return out
}
