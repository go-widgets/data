// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrKeyExists / ErrNotFound report the two identity failures a mutation can
// hit: inserting a key that already exists, or updating/deleting one that does
// not.
var (
	ErrKeyExists = errors.New("data: key already exists")
	ErrNotFound  = errors.New("data: record not found")
)

// MemoryProxy is an in-process Proxy over a slice of validated records. It is
// the reference backend: its Query runs the shared Apply engine directly, and a
// grpcproxy Server wraps one to serve the same results across the wire.
//
// It is safe for concurrent use.
type MemoryProxy struct {
	schema Schema
	key    string
	mu     sync.RWMutex
	rows   []Record
}

// NewMemoryProxy builds a MemoryProxy for the given schema, using keyField as
// the record identity (it must be a declared field). Each seed record is
// validated and must have a unique key. It returns an error on an unknown key
// field, an invalid seed, or a duplicate key.
func NewMemoryProxy(schema Schema, keyField string, seed ...Record) (*MemoryProxy, error) {
	if _, ok := schema.Field(keyField); !ok {
		return nil, fmt.Errorf("data: key field %q is not in the schema", keyField)
	}
	m := &MemoryProxy{schema: schema, key: keyField}
	for _, r := range seed {
		if err := m.insert(r); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// indexOf returns the position of the row whose key field equals key, or -1.
// Caller holds at least a read lock.
func (m *MemoryProxy) indexOf(key Value) int {
	for i, r := range m.rows {
		if r[m.key] == key {
			return i
		}
	}
	return -1
}

// insert validates r and appends it, rejecting a duplicate key. Caller holds the
// write lock (or is the constructor, which is not yet shared).
func (m *MemoryProxy) insert(r Record) error {
	if err := m.schema.Validate(r); err != nil {
		return err
	}
	if m.indexOf(r[m.key]) >= 0 {
		return ErrKeyExists
	}
	m.rows = append(m.rows, r.clone())
	return nil
}

// List returns a deep copy of every record.
func (m *MemoryProxy) List(_ context.Context) ([]Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneRows(m.rows), nil
}

// Query applies q via the shared engine over a snapshot of the records.
func (m *MemoryProxy) Query(_ context.Context, q Query) (View, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Apply(m.rows, q), nil
}

// Mutate applies one write under the write lock.
func (m *MemoryProxy) Mutate(_ context.Context, mut Mutation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch mut.Kind {
	case MutInsert:
		return m.insert(mut.Record)
	case MutUpdate:
		if err := m.schema.Validate(mut.Record); err != nil {
			return err
		}
		i := m.indexOf(mut.Record[m.key])
		if i < 0 {
			return ErrNotFound
		}
		m.rows[i] = mut.Record.clone()
		return nil
	default: // MutDelete
		i := m.indexOf(mut.Key)
		if i < 0 {
			return ErrNotFound
		}
		m.rows = append(m.rows[:i], m.rows[i+1:]...)
		return nil
	}
}
