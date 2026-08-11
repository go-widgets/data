// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import (
	"sort"
	"strconv"
	"strings"
)

// Canonical renders a View as a deterministic byte string: the same View value
// always yields the same bytes, and — crucially — a View computed by MemoryProxy
// and the identical View reconstructed from a grpcproxy round-trip encode to the
// SAME bytes. That is the equality the proxy-conformance test asserts.
//
// Determinism is achieved by never depending on Go map iteration order: record
// fields and aggregate labels are always emitted in sorted order, and rows and
// groups keep their query order. The format is unambiguous (length-tagged
// enough for the fields we carry), but it is meant for equality, not parsing.
func Canonical(v View) []byte {
	var b strings.Builder
	b.WriteString("total=")
	b.WriteString(strconv.Itoa(v.Total))
	b.WriteByte('\n')

	b.WriteString("aggs=")
	writeAggs(&b, v.Aggregates)
	b.WriteByte('\n')

	if v.Groups != nil {
		b.WriteString("groups=")
		b.WriteString(strconv.Itoa(len(v.Groups)))
		b.WriteByte('\n')
		for _, g := range v.Groups {
			b.WriteString("group key=")
			b.WriteString(g.Key.canonical())
			b.WriteString(" aggs=")
			writeAggs(&b, g.Aggregates)
			b.WriteByte('\n')
			writeRows(&b, g.Rows)
		}
		return []byte(b.String())
	}

	b.WriteString("rows=")
	b.WriteString(strconv.Itoa(len(v.Rows)))
	b.WriteByte('\n')
	writeRows(&b, v.Rows)
	return []byte(b.String())
}

// writeRows appends every row in order, each as its sorted-field canonical form.
func writeRows(b *strings.Builder, rows []Record) {
	for _, r := range rows {
		b.WriteString("  row")
		for _, name := range r.sortedNames() {
			b.WriteByte(' ')
			b.WriteString(name)
			b.WriteByte('=')
			b.WriteString(r[name].canonical())
		}
		b.WriteByte('\n')
	}
}

// writeAggs appends the aggregates in sorted-label order (or "-" when none), so
// the aggregate map never leaks its iteration order into the bytes.
func writeAggs(b *strings.Builder, aggs map[string]Value) {
	if len(aggs) == 0 {
		b.WriteByte('-')
		return
	}
	labels := make([]string, 0, len(aggs))
	for k := range aggs {
		labels = append(labels, k)
	}
	sort.Strings(labels)
	for i, k := range labels {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(aggs[k].canonical())
	}
}
