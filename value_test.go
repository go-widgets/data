// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import "testing"

func TestValueConstructorsAndNum(t *testing.T) {
	if v := String("hi"); v.Kind != KindString || v.Str != "hi" {
		t.Fatalf("String = %+v", v)
	}
	if v := Int(7); v.Kind != KindInt || v.Int != 7 || v.num() != 7 {
		t.Fatalf("Int = %+v num=%v", v, v.num())
	}
	if v := Float(1.5); v.Kind != KindFloat || v.Float != 1.5 || v.num() != 1.5 {
		t.Fatalf("Float = %+v", v)
	}
	if v := Bool(true); v.Kind != KindBool || !v.Bool || v.num() != 0 {
		t.Fatalf("Bool = %+v num=%v", v, v.num())
	}
	// num on a string is 0 (the default branch).
	if String("x").num() != 0 {
		t.Fatal("string num should be 0")
	}
}

func TestValueCompare(t *testing.T) {
	// Different kinds order by Kind.
	if String("z").compare(Int(1)) != -1 { // KindString(0) < KindInt(1)
		t.Fatal("cross-kind order")
	}
	if Int(1).compare(String("a")) != 1 {
		t.Fatal("cross-kind order reverse")
	}
	// Strings lexicographic.
	if String("a").compare(String("b")) != -1 || String("b").compare(String("a")) != 1 ||
		String("a").compare(String("a")) != 0 {
		t.Fatal("string compare")
	}
	// Numbers.
	if Int(1).compare(Int(2)) != -1 || Int(2).compare(Int(1)) != 1 || Int(2).compare(Int(2)) != 0 {
		t.Fatal("int compare")
	}
	if Float(1).compare(Float(2)) != -1 {
		t.Fatal("float compare")
	}
	// Bools: false < true.
	if Bool(false).compare(Bool(true)) != -1 || Bool(true).compare(Bool(false)) != 1 ||
		Bool(true).compare(Bool(true)) != 0 {
		t.Fatal("bool compare")
	}
}

func TestValueCanonical(t *testing.T) {
	cases := map[Value]string{
		String("hi"): "s:hi",
		Int(-3):      "i:-3",
		Float(1.5):   "f:1.5",
		Bool(true):   "b:true",
		Bool(false):  "b:false",
	}
	for v, want := range cases {
		if got := v.canonical(); got != want {
			t.Errorf("canonical(%+v) = %q, want %q", v, got, want)
		}
	}
}
