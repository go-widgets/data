// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import (
	"bytes"
	"testing"
)

func TestCanonicalIsOrderIndependent(t *testing.T) {
	// Two records with the same content built in different insertion orders must
	// canonicalise identically (fields are emitted sorted).
	a := Record{"b": Int(2), "a": String("x"), "c": Bool(true)}
	b := Record{"c": Bool(true), "a": String("x"), "b": Int(2)}
	va := Canonical(View{Rows: []Record{a}, Total: 1})
	vb := Canonical(View{Rows: []Record{b}, Total: 1})
	if !bytes.Equal(va, vb) {
		t.Fatalf("field order leaked:\n%s\n---\n%s", va, vb)
	}
}

func TestCanonicalUngroupedShape(t *testing.T) {
	v := View{
		Rows:       []Record{{"id": Int(1), "name": String("ann")}},
		Total:      1,
		Aggregates: map[string]Value{"count": Int(1), "sum(x)": Float(2.5)},
	}
	got := string(Canonical(v))
	want := "total=1\n" +
		"aggs=count=i:1,sum(x)=f:2.5\n" +
		"rows=1\n" +
		"  row id=i:1 name=s:ann\n"
	if got != want {
		t.Fatalf("canonical =\n%q\nwant\n%q", got, want)
	}
}

func TestCanonicalGroupedAndNoAggs(t *testing.T) {
	v := View{
		Total: 2,
		Groups: []Group{
			{Key: String("red"), Rows: []Record{{"id": Int(1)}}, Aggregates: map[string]Value{"count": Int(1)}},
			{Key: String("blue"), Rows: []Record{{"id": Int(2)}}, Aggregates: nil},
		},
	}
	got := string(Canonical(v))
	want := "total=2\n" +
		"aggs=-\n" +
		"groups=2\n" +
		"group key=s:red aggs=count=i:1\n" +
		"  row id=i:1\n" +
		"group key=s:blue aggs=-\n" +
		"  row id=i:2\n"
	if got != want {
		t.Fatalf("grouped canonical =\n%q\nwant\n%q", got, want)
	}
}
