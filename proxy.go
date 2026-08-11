// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import "context"

// Proxy is the pluggable data backend a Store talks to. The same three
// operations are served in-process by MemoryProxy and remotely by
// grpcproxy.Client, so a Store — and every sort/filter/group/page/aggregate it
// drives — is oblivious to whether the data is local or across a websocket.
type Proxy interface {
	// List returns every record, unfiltered and unordered (a fresh copy each
	// call, so the caller can retain it safely).
	List(ctx context.Context) ([]Record, error)
	// Query applies q to the backend's records and returns the resulting View.
	Query(ctx context.Context, q Query) (View, error)
	// Mutate inserts, updates or deletes one record and reports any validation
	// or not-found error.
	Mutate(ctx context.Context, m Mutation) error
}

// MutationKind is the operation a Mutation performs.
type MutationKind uint8

const (
	// MutInsert adds Record (which must be new under the key field).
	MutInsert MutationKind = iota
	// MutUpdate replaces the record whose key field equals Record's key field.
	MutUpdate
	// MutDelete removes the record whose key field equals Key.
	MutDelete
)

// Mutation is a single write. Insert and Update carry the full Record; Delete
// carries only the Key (the value of the backend's key field to remove).
type Mutation struct {
	Kind   MutationKind
	Record Record
	Key    Value
}
