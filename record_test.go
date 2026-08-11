// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import (
	"strings"
	"testing"
)

// testSchema is a small people schema reused across the package's tests.
func testSchema() Schema {
	return Schema{Fields: []Field{
		{Name: "id", Kind: KindInt},
		{Name: "name", Kind: KindString, Rules: []Rule{Required("name required"), StrMaxLen(10, "too long")}},
		{Name: "team", Kind: KindString},
		{Name: "salary", Kind: KindFloat, Rules: []Rule{NumMin(0, "no negative pay")}},
		{Name: "active", Kind: KindBool},
	}}
}

func rec(id int64, name, team string, salary float64, active bool) Record {
	return Record{
		"id": Int(id), "name": String(name), "team": String(team),
		"salary": Float(salary), "active": Bool(active),
	}
}

func TestRecordCloneIsIndependent(t *testing.T) {
	r := rec(1, "a", "x", 10, true)
	c := r.clone()
	c["name"] = String("mutated")
	if r["name"] != String("a") {
		t.Fatal("clone aliased original")
	}
}

func TestSchemaFieldLookup(t *testing.T) {
	s := testSchema()
	if f, ok := s.Field("name"); !ok || f.Kind != KindString {
		t.Fatalf("Field(name) = %+v ok=%v", f, ok)
	}
	if _, ok := s.Field("nope"); ok {
		t.Fatal("Field(nope) should miss")
	}
}

func TestSchemaValidate(t *testing.T) {
	s := testSchema()
	if err := s.Validate(rec(1, "ok", "x", 5, true)); err != nil {
		t.Fatalf("valid row rejected: %v", err)
	}

	// Missing field.
	miss := rec(1, "ok", "x", 5, true)
	delete(miss, "team")
	if err := s.Validate(miss); err == nil || !strings.Contains(err.Error(), "missing field") {
		t.Fatalf("missing: %v", err)
	}

	// Wrong kind.
	wrong := rec(1, "ok", "x", 5, true)
	wrong["id"] = String("nope")
	if err := s.Validate(wrong); err == nil || !strings.Contains(err.Error(), "want kind") {
		t.Fatalf("wrong kind: %v", err)
	}

	// Rule failure (Required on empty name).
	empty := rec(1, "", "x", 5, true)
	if err := s.Validate(empty); err == nil || !strings.Contains(err.Error(), "name required") {
		t.Fatalf("required: %v", err)
	}

	// Unknown field.
	extra := rec(1, "ok", "x", 5, true)
	extra["ghost"] = Int(9)
	if err := s.Validate(extra); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown: %v", err)
	}
}

func TestRules(t *testing.T) {
	// Required across kinds: zero of each kind fails, non-zero passes.
	req := Required("req")
	for _, z := range []Value{String(""), Int(0), Float(0), Bool(false)} {
		if req(z) == nil {
			t.Fatalf("Required should reject zero %+v", z)
		}
	}
	for _, nz := range []Value{String("x"), Int(1), Float(0.1), Bool(true)} {
		if req(nz) != nil {
			t.Fatalf("Required should accept %+v", nz)
		}
	}

	if StrMinLen(3, "m")(String("ab")) == nil || StrMinLen(3, "m")(String("abc")) != nil {
		t.Fatal("StrMinLen")
	}
	if StrMaxLen(2, "m")(String("abc")) == nil || StrMaxLen(2, "m")(String("ab")) != nil {
		t.Fatal("StrMaxLen")
	}
	if NumMin(0, "m")(Float(-1)) == nil || NumMin(0, "m")(Float(0)) != nil {
		t.Fatal("NumMin")
	}
	if NumMax(10, "m")(Int(11)) == nil || NumMax(10, "m")(Int(10)) != nil {
		t.Fatal("NumMax")
	}
}
