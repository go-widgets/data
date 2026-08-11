// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import (
	"context"
	"errors"
	"testing"
)

func mustMemory(t *testing.T, seed ...Record) *MemoryProxy {
	t.Helper()
	m, err := NewMemoryProxy(testSchema(), "id", seed...)
	if err != nil {
		t.Fatalf("NewMemoryProxy: %v", err)
	}
	return m
}

func TestNewMemoryProxyGuards(t *testing.T) {
	// Unknown key field.
	if _, err := NewMemoryProxy(testSchema(), "nope"); err == nil {
		t.Fatal("expected unknown-key error")
	}
	// Invalid seed (Required name empty).
	if _, err := NewMemoryProxy(testSchema(), "id", rec(1, "", "x", 1, true)); err == nil {
		t.Fatal("expected invalid-seed error")
	}
	// Duplicate seed key.
	_, err := NewMemoryProxy(testSchema(), "id",
		rec(1, "a", "x", 1, true), rec(1, "b", "y", 2, true))
	if !errors.Is(err, ErrKeyExists) {
		t.Fatalf("dup seed err = %v", err)
	}
}

func TestMemoryListAndQuery(t *testing.T) {
	m := mustMemory(t, sampleRows()...)
	ctx := context.Background()

	list, err := m.List(ctx)
	if err != nil || len(list) != 5 {
		t.Fatalf("List = %d rows, err %v", len(list), err)
	}
	// List returns copies: mutating one must not affect a later List.
	list[0]["name"] = String("hax")
	again, _ := m.List(ctx)
	if again[0]["name"] == String("hax") {
		t.Fatal("List aliased internal storage")
	}

	v, err := m.Query(ctx, Query{Filters: []Filter{{"team", OpEq, String("red")}}})
	if err != nil || v.Total != 3 {
		t.Fatalf("Query = %+v err %v", v.Total, err)
	}
}

func TestMemoryMutate(t *testing.T) {
	m := mustMemory(t, rec(1, "a", "x", 1, true))
	ctx := context.Background()

	// Insert new.
	if err := m.Mutate(ctx, Mutation{Kind: MutInsert, Record: rec(2, "b", "y", 2, true)}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// Insert duplicate key → ErrKeyExists.
	if err := m.Mutate(ctx, Mutation{Kind: MutInsert, Record: rec(2, "c", "z", 3, true)}); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("dup insert err = %v", err)
	}
	// Insert invalid → validation error.
	if err := m.Mutate(ctx, Mutation{Kind: MutInsert, Record: rec(9, "", "z", 3, true)}); err == nil {
		t.Fatal("invalid insert should fail")
	}

	// Update existing.
	if err := m.Mutate(ctx, Mutation{Kind: MutUpdate, Record: rec(2, "bee", "y", 22, true)}); err != nil {
		t.Fatalf("update: %v", err)
	}
	v, _ := m.Query(ctx, Query{Filters: []Filter{{"id", OpEq, Int(2)}}})
	if v.Rows[0]["name"] != String("bee") || v.Rows[0]["salary"] != Float(22) {
		t.Fatalf("update not applied: %+v", v.Rows[0])
	}
	// Update missing key → ErrNotFound.
	if err := m.Mutate(ctx, Mutation{Kind: MutUpdate, Record: rec(99, "x", "y", 1, true)}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing err = %v", err)
	}
	// Update invalid → validation error (before the not-found check).
	if err := m.Mutate(ctx, Mutation{Kind: MutUpdate, Record: rec(2, "", "y", 1, true)}); err == nil {
		t.Fatal("invalid update should fail")
	}

	// Delete existing.
	if err := m.Mutate(ctx, Mutation{Kind: MutDelete, Key: Int(1)}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// Delete missing → ErrNotFound.
	if err := m.Mutate(ctx, Mutation{Kind: MutDelete, Key: Int(1)}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing err = %v", err)
	}
	list, _ := m.List(ctx)
	if len(list) != 1 || list[0]["id"] != Int(2) {
		t.Fatalf("after deletes = %+v", list)
	}
}
