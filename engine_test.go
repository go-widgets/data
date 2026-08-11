// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import "testing"

// sample rows shared by the engine tests.
func sampleRows() []Record {
	return []Record{
		rec(1, "ann", "red", 30, true),
		rec(2, "bob", "blue", 20, false),
		rec(3, "cara", "red", 50, true),
		rec(4, "dan", "blue", 40, true),
		rec(5, "eve", "red", 10, false),
	}
}

func TestApplyFilterOps(t *testing.T) {
	rows := sampleRows()
	tests := []struct {
		name   string
		filter Filter
		want   int
	}{
		{"eq", Filter{"team", OpEq, String("red")}, 3},
		{"ne", Filter{"team", OpNe, String("red")}, 2},
		{"lt", Filter{"salary", OpLt, Float(30)}, 2},
		{"le", Filter{"salary", OpLe, Float(30)}, 3},
		{"gt", Filter{"salary", OpGt, Float(30)}, 2},
		{"ge", Filter{"salary", OpGe, Float(30)}, 3},
		{"contains", Filter{"name", OpContains, String("a")}, 3}, // ann, cara, dan
		{"contains-nonstring-cell", Filter{"id", OpContains, String("1")}, 0},
		{"missing-field", Filter{"ghost", OpEq, Int(1)}, 0},
	}
	for _, tc := range tests {
		v := Apply(rows, Query{Filters: []Filter{tc.filter}})
		if v.Total != tc.want {
			t.Errorf("%s: Total = %d, want %d", tc.name, v.Total, tc.want)
		}
	}
}

func TestApplyFilterAndedAndContainsEmpty(t *testing.T) {
	rows := sampleRows()
	v := Apply(rows, Query{Filters: []Filter{
		{"team", OpEq, String("red")},
		{"active", OpEq, Bool(true)},
	}})
	if v.Total != 2 { // ann, cara
		t.Fatalf("ANDed = %d, want 2", v.Total)
	}
	// Empty substring matches every string cell.
	if got := Apply(rows, Query{Filters: []Filter{{"name", OpContains, String("")}}}).Total; got != 5 {
		t.Fatalf("empty contains = %d, want 5", got)
	}
}

func TestApplySortAscDescMultiAndMissing(t *testing.T) {
	rows := sampleRows()
	asc := Apply(rows, Query{Sorts: []Sort{{Field: "salary"}}})
	if asc.Rows[0]["name"] != String("eve") || asc.Rows[4]["name"] != String("cara") {
		t.Fatalf("asc order wrong: %v .. %v", asc.Rows[0]["name"], asc.Rows[4]["name"])
	}
	desc := Apply(rows, Query{Sorts: []Sort{{Field: "salary", Desc: true}}})
	if desc.Rows[0]["name"] != String("cara") {
		t.Fatalf("desc order wrong: %v", desc.Rows[0]["name"])
	}
	// Multi-key: team asc, then salary desc within team.
	multi := Apply(rows, Query{Sorts: []Sort{{Field: "team"}, {Field: "salary", Desc: true}}})
	// blue: dan(40), bob(20); red: cara(50), ann(30), eve(10)
	order := []string{"dan", "bob", "cara", "ann", "eve"}
	for i, want := range order {
		if multi.Rows[i]["name"] != String(want) {
			t.Fatalf("multi[%d] = %v, want %s", i, multi.Rows[i]["name"], want)
		}
	}
	// A sort on a field some rows lack must not panic and stays stable.
	mixed := []Record{{"id": Int(1), "k": String("b")}, {"id": Int(2)}, {"id": Int(3), "k": String("a")}}
	_ = Apply(mixed, Query{Sorts: []Sort{{Field: "k"}}})
}

func TestApplyPagingUngrouped(t *testing.T) {
	rows := sampleRows()
	q := Query{Sorts: []Sort{{Field: "id"}}, Offset: 1, Limit: 2}
	v := Apply(rows, q)
	if v.Total != 5 || len(v.Rows) != 2 ||
		v.Rows[0]["id"] != Int(2) || v.Rows[1]["id"] != Int(3) {
		t.Fatalf("page = %+v", v.Rows)
	}
	// Offset past the end → empty page, Total intact.
	if got := Apply(rows, Query{Offset: 99}); len(got.Rows) != 0 || got.Total != 5 {
		t.Fatalf("over-offset = %+v", got)
	}
	// Negative offset clamps to 0; zero limit = all.
	if got := Apply(rows, Query{Offset: -5}); len(got.Rows) != 5 {
		t.Fatalf("neg offset = %d rows", len(got.Rows))
	}
}

func TestApplyGroupingAndGroupPaging(t *testing.T) {
	rows := sampleRows()
	q := Query{
		Sorts:   []Sort{{Field: "team"}, {Field: "id"}},
		GroupBy: "team",
		Aggs:    []Agg{{Func: AggCount}, {Field: "salary", Func: AggSum}},
	}
	v := Apply(rows, q)
	if v.Rows != nil {
		t.Fatal("grouped view should have nil Rows")
	}
	if len(v.Groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(v.Groups))
	}
	// blue first (sorted), count 2, sum 60.
	blue := v.Groups[0]
	if blue.Key != String("blue") || blue.Aggregates["count"] != Int(2) ||
		blue.Aggregates["sum(salary)"] != Float(60) {
		t.Fatalf("blue group = %+v", blue)
	}
	// Grand aggregates over the whole filtered set.
	if v.Aggregates["count"] != Int(5) || v.Aggregates["sum(salary)"] != Float(150) {
		t.Fatalf("grand aggs = %+v", v.Aggregates)
	}
	// Paginate groups: Limit 1 → only the first group.
	one := Apply(rows, Query{GroupBy: "team", Sorts: []Sort{{Field: "team"}}, Limit: 1})
	if len(one.Groups) != 1 || one.Groups[0].Key != String("blue") {
		t.Fatalf("group page = %+v", one.Groups)
	}
}

func TestReduceAllFuncsAndEmpty(t *testing.T) {
	rows := sampleRows()
	all := Apply(rows, Query{Aggs: []Agg{
		{Func: AggCount},
		{Field: "salary", Func: AggSum},
		{Field: "salary", Func: AggAvg},
		{Field: "salary", Func: AggMin},
		{Field: "salary", Func: AggMax},
	}})
	a := all.Aggregates
	if a["count"] != Int(5) || a["sum(salary)"] != Float(150) || a["avg(salary)"] != Float(30) ||
		a["min(salary)"] != Float(10) || a["max(salary)"] != Float(50) {
		t.Fatalf("aggs = %+v", a)
	}
	// Empty set: count 0, sum/avg 0, min/max zero Value.
	empty := Apply(nil, Query{Aggs: []Agg{
		{Func: AggCount}, {Field: "salary", Func: AggSum},
		{Field: "salary", Func: AggAvg}, {Field: "salary", Func: AggMin},
	}})
	e := empty.Aggregates
	if e["count"] != Int(0) || e["sum(salary)"] != Float(0) ||
		e["avg(salary)"] != Float(0) || e["min(salary)"] != (Value{}) {
		t.Fatalf("empty aggs = %+v", e)
	}
	// A reduce over rows that lack the field skips them (missing-cell branch).
	noField := []Record{{"id": Int(1)}, {"id": Int(2)}}
	got := Apply(noField, Query{Aggs: []Agg{{Field: "salary", Func: AggMax}}})
	if got.Aggregates["max(salary)"] != (Value{}) {
		t.Fatalf("max over missing = %+v", got.Aggregates)
	}
	// No aggregates → nil map.
	if Apply(rows, Query{}).Aggregates != nil {
		t.Fatal("no-agg view should have nil Aggregates")
	}
}

func TestApplyDoesNotMutateOrAliasInput(t *testing.T) {
	rows := sampleRows()
	v := Apply(rows, Query{Sorts: []Sort{{Field: "salary"}}})
	// Mutating a returned row must not touch the source (cloneRows).
	v.Rows[0]["name"] = String("changed")
	for _, r := range rows {
		if r["name"] == String("changed") {
			t.Fatal("Apply aliased its input rows")
		}
	}
}

func TestAggLabels(t *testing.T) {
	cases := map[Agg]string{
		{Func: AggCount}:           "count",
		{Field: "p", Func: AggSum}: "sum(p)",
		{Field: "p", Func: AggAvg}: "avg(p)",
		{Field: "q", Func: AggMin}: "min(q)",
		{Field: "q", Func: AggMax}: "max(q)",
	}
	for a, want := range cases {
		if got := a.label(); got != want {
			t.Errorf("label(%+v) = %q, want %q", a, got, want)
		}
	}
}
