// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

// FilterOp is a comparison a Filter applies between a record's cell and a
// reference Value.
type FilterOp uint8

const (
	// OpEq keeps rows whose cell equals the reference value.
	OpEq FilterOp = iota
	// OpNe keeps rows whose cell differs from the reference value.
	OpNe
	// OpLt / OpLe / OpGt / OpGe keep rows ordered below / at-or-below / above /
	// at-or-above the reference value (Value.compare ordering).
	OpLt
	OpLe
	OpGt
	OpGe
	// OpContains keeps rows whose string cell contains the reference substring.
	// It only matches string cells (a non-string cell never contains).
	OpContains
)

// Filter is one predicate: keep the rows for which Field's cell relates to Value
// under Op. A filter on a field a row lacks never matches.
type Filter struct {
	Field string
	Op    FilterOp
	Value Value
}

// Sort is one ordering key: order by Field ascending, or descending when Desc.
// A Query's Sorts apply in order, the first being the primary key.
type Sort struct {
	Field string
	Desc  bool
}

// AggFunc is a column aggregation.
type AggFunc uint8

const (
	// AggCount counts rows (its Field is ignored).
	AggCount AggFunc = iota
	// AggSum / AggAvg / AggMin / AggMax reduce a numeric column.
	AggSum
	AggAvg
	AggMin
	AggMax
)

// Agg requests one aggregate over a column. For AggCount the Field is ignored.
type Agg struct {
	Field string
	Func  AggFunc
}

// label is the stable key an aggregate is stored under in a View — "count",
// "sum(price)", "avg(price)", "min(qty)", "max(qty)" — so the aggregates map is
// deterministic and self-describing.
func (a Agg) label() string {
	switch a.Func {
	case AggCount:
		return "count"
	case AggSum:
		return "sum(" + a.Field + ")"
	case AggAvg:
		return "avg(" + a.Field + ")"
	case AggMin:
		return "min(" + a.Field + ")"
	default: // AggMax
		return "max(" + a.Field + ")"
	}
}

// Query is a full read specification the engine applies to a set of records:
// keep the rows matching every Filter, order them by Sorts, optionally group by
// a field, take a page (Offset/Limit), and compute Aggs. Zero-value fields are
// inert — an empty Query returns every row unchanged with no grouping or paging.
type Query struct {
	// Filters are ANDed: a row must satisfy all of them.
	Filters []Filter
	// Sorts order the surviving rows (primary key first).
	Sorts []Sort
	// GroupBy, when non-empty, groups the ordered rows by that field's value.
	GroupBy string
	// Offset skips this many rows (ungrouped) or groups (grouped) before the page.
	Offset int
	// Limit caps the page to this many rows (ungrouped) or groups (grouped); 0
	// means no limit.
	Limit int
	// Aggs are computed over the whole filtered set (View.Aggregates) and, when
	// grouped, over each group (Group.Aggregates).
	Aggs []Agg
}

// Group is one bucket of a grouped View: the shared key value, the group's rows
// (in the query's sort order), and its per-group aggregates.
type Group struct {
	Key        Value
	Rows       []Record
	Aggregates map[string]Value
}

// View is the result of applying a Query. When the query is ungrouped, Rows is
// the requested page and Groups is nil; when it is grouped, Groups is the page
// of groups and Rows is nil. Total is the number of rows that matched the filter
// (before paging), and Aggregates holds the query's aggregates over that whole
// filtered set.
type View struct {
	Rows       []Record
	Groups     []Group
	Total      int
	Aggregates map[string]Value
}

// pageBounds clamps an Offset/Limit against a collection of n items and returns
// the [lo, hi) slice bounds of the page. A non-positive Limit means "to the
// end"; an Offset past the end yields an empty page.
func pageBounds(n, offset, limit int) (lo, hi int) {
	if offset < 0 {
		offset = 0
	}
	if offset > n {
		offset = n
	}
	hi = n
	if limit > 0 && offset+limit < n {
		hi = offset + limit
	}
	return offset, hi
}
