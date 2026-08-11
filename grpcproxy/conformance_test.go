// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package grpcproxy

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-widgets/data"
)

// confSchema / confRows are the fixture the conformance battery runs against.
func confSchema() data.Schema {
	return data.Schema{Fields: []data.Field{
		{Name: "id", Kind: data.KindInt},
		{Name: "name", Kind: data.KindString},
		{Name: "team", Kind: data.KindString},
		{Name: "salary", Kind: data.KindFloat},
		{Name: "active", Kind: data.KindBool},
	}}
}

func confRow(id int64, name, team string, salary float64, active bool) data.Record {
	return data.Record{
		"id": data.Int(id), "name": data.String(name), "team": data.String(team),
		"salary": data.Float(salary), "active": data.Bool(active),
	}
}

func confRows() []data.Record {
	return []data.Record{
		confRow(1, "ann", "red", 30, true),
		confRow(2, "bob", "blue", 20, false),
		confRow(3, "cara", "red", 50, true),
		confRow(4, "dan", "blue", 40, true),
		confRow(5, "eve", "red", 10, false),
		confRow(6, "finn", "green", 40, true),
	}
}

// conformanceQueries is the battery of sort→filter→group→page→aggregate
// combinations the two proxies must agree on, byte for byte.
func conformanceQueries() map[string]data.Query {
	allAggs := []data.Agg{
		{Func: data.AggCount},
		{Field: "salary", Func: data.AggSum},
		{Field: "salary", Func: data.AggAvg},
		{Field: "salary", Func: data.AggMin},
		{Field: "salary", Func: data.AggMax},
	}
	return map[string]data.Query{
		"empty":       {},
		"sort-asc":    {Sorts: []data.Sort{{Field: "salary"}}},
		"sort-desc":   {Sorts: []data.Sort{{Field: "salary", Desc: true}}},
		"sort-multi":  {Sorts: []data.Sort{{Field: "team"}, {Field: "salary", Desc: true}}},
		"filter-team": {Filters: []data.Filter{{Field: "team", Op: data.OpEq, Value: data.String("red")}}},
		"filter-num":  {Filters: []data.Filter{{Field: "salary", Op: data.OpGe, Value: data.Float(30)}}},
		"filter-contains": {
			Filters: []data.Filter{{Field: "name", Op: data.OpContains, Value: data.String("a")}},
			Sorts:   []data.Sort{{Field: "id"}},
		},
		"page": {Sorts: []data.Sort{{Field: "id"}}, Offset: 1, Limit: 3},
		"filter-sort-page": {
			Filters: []data.Filter{{Field: "active", Op: data.OpEq, Value: data.Bool(true)}},
			Sorts:   []data.Sort{{Field: "salary", Desc: true}},
			Offset:  1, Limit: 2,
		},
		"group": {
			Sorts: []data.Sort{{Field: "team"}, {Field: "id"}}, GroupBy: "team", Aggs: allAggs,
		},
		"group-page": {
			Sorts: []data.Sort{{Field: "team"}}, GroupBy: "team", Offset: 1, Limit: 1, Aggs: allAggs,
		},
		"full-pipeline": {
			Filters: []data.Filter{{Field: "salary", Op: data.OpGe, Value: data.Float(20)}},
			Sorts:   []data.Sort{{Field: "team"}, {Field: "salary", Desc: true}},
			GroupBy: "team", Aggs: allAggs,
		},
		"grand-aggs-only": {Aggs: allAggs},
		"empty-result":    {Filters: []data.Filter{{Field: "team", Op: data.OpEq, Value: data.String("nope")}}, Aggs: allAggs},
		"empty-group":     {GroupBy: "team", Filters: []data.Filter{{Field: "id", Op: data.OpEq, Value: data.Int(999)}}, Aggs: allAggs},
	}
}

// TestProxyConformance is the crown-jewel assertion: for every query in the
// battery, the in-process MemoryProxy and the remote gRPC Client must return
// Views that canonicalise to identical bytes — and both must equal what the
// shared engine computes directly. If they ever diverge, the same data code
// would render differently native vs in a browser, and this fails loudly.
func TestProxyConformance(t *testing.T) {
	ctx := context.Background()
	mem, err := data.NewMemoryProxy(confSchema(), "id", confRows()...)
	if err != nil {
		t.Fatalf("memory proxy: %v", err)
	}
	url, stop, err := Serve(":0", mem)
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	defer stop()
	client, err := Dial(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	queries := conformanceQueries()
	total := 0
	for name, q := range queries {
		local, err := mem.Query(ctx, q)
		if err != nil {
			t.Fatalf("%s: local query: %v", name, err)
		}
		remote, err := client.Query(ctx, q)
		if err != nil {
			t.Fatalf("%s: remote query: %v", name, err)
		}
		reference := data.Apply(confRows(), q)

		cl := data.Canonical(local)
		cr := data.Canonical(remote)
		cref := data.Canonical(reference)
		if !bytes.Equal(cl, cr) {
			t.Fatalf("%s: MemoryProxy vs GRPCProxy DIVERGE:\n--- memory ---\n%s\n--- grpc ---\n%s",
				name, cl, cr)
		}
		if !bytes.Equal(cl, cref) {
			t.Fatalf("%s: proxy vs reference engine diverge:\n%s\n---\n%s", name, cl, cref)
		}
		total++
	}
	t.Logf("proxy conformance: %d queries, MemoryProxy == GRPCProxy == engine (byte-identical)", total)
}
