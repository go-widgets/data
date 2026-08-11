// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import (
	"errors"
	"fmt"
	"sort"
)

// Record is one row: a set of named typed cells. It is a plain map so callers
// build and read rows with ordinary Go, while the Schema gives it type and the
// canonical encoder gives it a stable serialization (fields are always emitted
// in sorted-name order, so map iteration order never leaks into a comparison).
type Record map[string]Value

// clone returns an independent copy so a proxy never hands out its internal row.
func (r Record) clone() Record {
	out := make(Record, len(r))
	for k, v := range r {
		out[k] = v
	}
	return out
}

// sortedNames returns the record's field names in ascending order — the order
// the canonical encoder walks, and a convenient deterministic iteration order.
func (r Record) sortedNames() []string {
	names := make([]string, 0, len(r))
	for k := range r {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Rule validates one cell value, returning a non-nil error (whose text is the
// message shown to the user) when the value fails. It mirrors the string-rule
// shape of go-widgets/toolkit's validation, lifted to a typed Value so a schema
// can validate numbers and booleans, not only text.
type Rule func(Value) error

// Required rejects a zero value for the field's kind: an empty string, a zero
// number, or false. Use it to demand a present, non-default cell.
func Required(msg string) Rule {
	return func(v Value) error {
		zero := v == Value{Kind: v.Kind}
		if zero {
			return errors.New(msg)
		}
		return nil
	}
}

// StrMinLen rejects a string value shorter than n runes.
func StrMinLen(n int, msg string) Rule {
	return func(v Value) error {
		if len([]rune(v.Str)) < n {
			return errors.New(msg)
		}
		return nil
	}
}

// StrMaxLen rejects a string value longer than n runes.
func StrMaxLen(n int, msg string) Rule {
	return func(v Value) error {
		if len([]rune(v.Str)) > n {
			return errors.New(msg)
		}
		return nil
	}
}

// NumMin rejects a numeric value below lo (int and float compared as float64).
func NumMin(lo float64, msg string) Rule {
	return func(v Value) error {
		if v.num() < lo {
			return errors.New(msg)
		}
		return nil
	}
}

// NumMax rejects a numeric value above hi.
func NumMax(hi float64, msg string) Rule {
	return func(v Value) error {
		if v.num() > hi {
			return errors.New(msg)
		}
		return nil
	}
}

// Field declares one column of a Schema: its name, its scalar Kind, and any
// validation Rules run against a row's value for it.
type Field struct {
	Name  string
	Kind  Kind
	Rules []Rule
}

// Schema is an ordered list of Fields — the typed shape a Record must satisfy.
type Schema struct {
	Fields []Field
}

// Field looks a field up by name.
func (s Schema) Field(name string) (Field, bool) {
	for _, f := range s.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

// Validate checks a Record against the schema: every declared field must be
// present with the declared Kind and must pass its Rules, and the record must
// carry no field the schema does not declare. It returns the first violation, so
// the message is the one to surface. A nil error means the row is well-formed.
func (s Schema) Validate(r Record) error {
	declared := make(map[string]struct{}, len(s.Fields))
	for _, f := range s.Fields {
		declared[f.Name] = struct{}{}
		v, ok := r[f.Name]
		if !ok {
			return fmt.Errorf("missing field %q", f.Name)
		}
		if v.Kind != f.Kind {
			return fmt.Errorf("field %q: want kind %d, got %d", f.Name, f.Kind, v.Kind)
		}
		for _, rule := range f.Rules {
			if err := rule(v); err != nil {
				return fmt.Errorf("field %q: %w", f.Name, err)
			}
		}
	}
	for _, name := range r.sortedNames() {
		if _, ok := declared[name]; !ok {
			return fmt.Errorf("unknown field %q", name)
		}
	}
	return nil
}
