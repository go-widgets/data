// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import (
	"context"
	"errors"
	"testing"
)

// person is a typed row the Store maps to and from Records via a Codec.
type person struct {
	ID     int64
	Name   string
	Team   string
	Salary float64
	Active bool
}

func personCodec() Codec[person] {
	return Codec[person]{
		Encode: func(p person) Record {
			return rec(p.ID, p.Name, p.Team, p.Salary, p.Active)
		},
		Decode: func(r Record) person {
			return person{
				ID: r["id"].Int, Name: r["name"].Str, Team: r["team"].Str,
				Salary: r["salary"].Float, Active: r["active"].Bool,
			}
		},
	}
}

func TestStoreLoadUngroupedMirrorsList(t *testing.T) {
	m := mustMemory(t, sampleRows()...)
	s := NewStore(m, personCodec())
	s.SetQuery(Query{Sorts: []Sort{{Field: "salary", Desc: true}}, Limit: 2})
	if got := s.Query().Limit; got != 2 {
		t.Fatalf("Query() = %+v", s.Query())
	}

	// Track observable notifications so we prove the list drives a view.
	var changes int
	unsub := s.Items().SubscribeChanged(func() { changes++ })
	defer unsub()

	view, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if view.Total != 5 || s.Items().Len() != 2 {
		t.Fatalf("view.Total=%d items=%d", view.Total, s.Items().Len())
	}
	if s.Items().At(0).Name != "cara" || s.Items().At(1).Name != "dan" {
		t.Fatalf("items = %v, %v", s.Items().At(0), s.Items().At(1))
	}
	if changes == 0 {
		t.Fatal("Items never notified")
	}
}

func TestStoreLoadGroupedFlattens(t *testing.T) {
	m := mustMemory(t, sampleRows()...)
	s := NewStore(m, personCodec())
	s.SetQuery(Query{GroupBy: "team", Sorts: []Sort{{Field: "team"}, {Field: "id"}}})
	view, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(view.Groups) != 2 {
		t.Fatalf("groups = %d", len(view.Groups))
	}
	// Flattened items = all rows in group-then-row order: blue(bob,dan), red(ann,cara,eve).
	want := []string{"bob", "dan", "ann", "cara", "eve"}
	if s.Items().Len() != len(want) {
		t.Fatalf("items len = %d, want %d", s.Items().Len(), len(want))
	}
	for i, n := range want {
		if s.Items().At(i).Name != n {
			t.Fatalf("item[%d] = %s, want %s", i, s.Items().At(i).Name, n)
		}
	}
}

func TestStoreMutators(t *testing.T) {
	m := mustMemory(t, rec(1, "a", "x", 1, true))
	s := NewStore(m, personCodec())
	s.SetQuery(Query{Sorts: []Sort{{Field: "id"}}})
	ctx := context.Background()

	if err := s.Add(ctx, person{ID: 2, Name: "b", Team: "y", Salary: 2, Active: true}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if s.Items().Len() != 2 { // reload happened
		t.Fatalf("after Add items = %d", s.Items().Len())
	}
	if err := s.Update(ctx, person{ID: 2, Name: "bee", Team: "y", Salary: 3, Active: true}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if s.Items().At(1).Name != "bee" {
		t.Fatalf("update not reflected: %v", s.Items().At(1))
	}
	if err := s.Delete(ctx, Int(1)); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s.Items().Len() != 1 || s.Items().At(0).ID != 2 {
		t.Fatalf("after Delete items = %v", s.Items().Slice())
	}

	// Mutation error path: duplicate insert propagates and does not reload.
	if err := s.Add(ctx, person{ID: 2, Name: "dup", Team: "y", Salary: 1, Active: true}); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("dup Add err = %v", err)
	}
}

// failProxy makes Query fail, to cover Store.Load's error path.
type failProxy struct{ Proxy }

func (failProxy) Query(context.Context, Query) (View, error) {
	return View{}, errors.New("boom")
}

func TestStoreLoadErrorLeavesItems(t *testing.T) {
	m := mustMemory(t, rec(1, "a", "x", 1, true))
	s := NewStore[person](failProxy{Proxy: m}, personCodec())
	s.Items().Append(person{ID: 7, Name: "keep"})
	if _, err := s.Load(context.Background()); err == nil {
		t.Fatal("expected Load error")
	}
	if s.Items().Len() != 1 || s.Items().At(0).Name != "keep" {
		t.Fatalf("items disturbed on error: %v", s.Items().Slice())
	}
}
