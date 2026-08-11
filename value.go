// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package data is the headless data spine of the go-widgets ecosystem: a typed
// record model with validation, a collection with sort/filter/group/pagination/
// aggregation, and a pluggable proxy so the very same query pipeline runs
// in-process (MemoryProxy) or against a remote service (the grpcproxy
// subpackage) — natively and in a browser/wasm build alike. It imports only the
// standard library and go-widgets/mvvm, so nothing here depends on a GUI.
package data

import "strconv"

// Kind is the scalar type of a Value. The set is deliberately small — the four
// types a data grid needs — so a Value stays comparable and round-trips through
// the wire codec without loss.
type Kind uint8

const (
	// KindString is a UTF-8 text value.
	KindString Kind = iota
	// KindInt is a signed 64-bit integer value.
	KindInt
	// KindFloat is a 64-bit IEEE-754 float value.
	KindFloat
	// KindBool is a boolean value.
	KindBool
)

// Value is one typed scalar cell. It is a tagged union kept comparable (no
// slices or maps) so it can be a map key, sorted, and compared by ==. Only the
// field selected by Kind is meaningful; the others hold their zero value.
type Value struct {
	Kind  Kind
	Str   string
	Int   int64
	Float float64
	Bool  bool
}

// String makes a KindString Value.
func String(s string) Value { return Value{Kind: KindString, Str: s} }

// Int makes a KindInt Value.
func Int(i int64) Value { return Value{Kind: KindInt, Int: i} }

// Float makes a KindFloat Value.
func Float(f float64) Value { return Value{Kind: KindFloat, Float: f} }

// Bool makes a KindBool Value.
func Bool(b bool) Value { return Value{Kind: KindBool, Bool: b} }

// num reports the Value as a float64 for numeric aggregation and comparison: the
// int or float payload for the numeric kinds, else 0. Callers gate on Kind
// before relying on it for non-numeric values.
func (v Value) num() float64 {
	switch v.Kind {
	case KindInt:
		return float64(v.Int)
	case KindFloat:
		return v.Float
	default:
		return 0
	}
}

// compare orders two Values totally and deterministically. Values of different
// kinds order by Kind first (so a mixed column still has a stable order); within
// a kind they order naturally — lexicographically for strings, numerically for
// int/float (compared as float64 so an int and a float column sort sensibly),
// and false<true for bools. Returns -1, 0 or +1.
func (v Value) compare(o Value) int {
	if v.Kind != o.Kind {
		return cmpInt(int(v.Kind), int(o.Kind))
	}
	switch v.Kind {
	case KindString:
		return cmpStr(v.Str, o.Str)
	case KindBool:
		return cmpInt(b2i(v.Bool), b2i(o.Bool))
	default: // KindInt, KindFloat
		return cmpFloat(v.num(), o.num())
	}
}

// canonical renders the Value as a stable, kind-tagged string used by the View
// canonical encoder so two proxies that computed the same View produce identical
// bytes. Floats use the shortest round-trippable form, identical on both sides.
func (v Value) canonical() string {
	switch v.Kind {
	case KindString:
		return "s:" + v.Str
	case KindInt:
		return "i:" + strconv.FormatInt(v.Int, 10)
	case KindFloat:
		return "f:" + strconv.FormatFloat(v.Float, 'g', -1, 64)
	default: // KindBool
		return "b:" + strconv.FormatBool(v.Bool)
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func cmpStr(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
