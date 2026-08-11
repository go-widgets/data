// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package data

import (
	"context"

	"github.com/go-widgets/mvvm"
)

// Codec converts between a caller's typed row R and the schema-typed Record the
// proxy stores. Supplying the two functions keeps Store reflection-free and lets
// the app choose exactly how its struct maps onto fields.
type Codec[R any] struct {
	Encode func(R) Record
	Decode func(Record) R
}

// Store is the typed, bindable collection at the top of the spine. It holds a
// Query, runs it against a Proxy (local or remote — Store neither knows nor
// cares), and mirrors the resulting page of rows into an mvvm.ObservableList[R]
// so a view re-renders itself when the data changes. Sort/filter/group/page/
// aggregate all live in the Query; Load re-materialises the list from the proxy.
type Store[R any] struct {
	proxy Proxy
	codec Codec[R]
	query Query
	items *mvvm.ObservableList[R]
}

// NewStore builds a Store over proxy using codec to (de)serialise rows. Its
// Query starts empty (every row, no grouping/paging); set one with SetQuery.
func NewStore[R any](proxy Proxy, codec Codec[R]) *Store[R] {
	return &Store[R]{proxy: proxy, codec: codec, items: mvvm.NewObservableList[R]()}
}

// Items is the observable list a view binds to; it holds the decoded rows of the
// last Load (the flattened group rows when the query groups).
func (s *Store[R]) Items() *mvvm.ObservableList[R] { return s.items }

// Query returns the current query.
func (s *Store[R]) Query() Query { return s.query }

// SetQuery replaces the query used by the next Load.
func (s *Store[R]) SetQuery(q Query) { s.query = q }

// Load runs the current query against the proxy, resets Items to the decoded page
// rows (group rows in order when grouped), and returns the full View so a caller
// can also read Total, Groups and Aggregates. On a proxy error Items is left
// unchanged.
func (s *Store[R]) Load(ctx context.Context) (View, error) {
	view, err := s.proxy.Query(ctx, s.query)
	if err != nil {
		return View{}, err
	}
	rows := view.Rows
	if view.Groups != nil {
		rows = nil
		for _, g := range view.Groups {
			rows = append(rows, g.Rows...)
		}
	}
	decoded := make([]R, len(rows))
	for i, r := range rows {
		decoded[i] = s.codec.Decode(r)
	}
	s.items.Clear()
	s.items.Append(decoded...)
	return view, nil
}

// Add inserts a typed row, then reloads so Items reflects the new data through
// the current query.
func (s *Store[R]) Add(ctx context.Context, row R) error {
	return s.mutateThenLoad(ctx, Mutation{Kind: MutInsert, Record: s.codec.Encode(row)})
}

// Update replaces the row sharing this row's key, then reloads.
func (s *Store[R]) Update(ctx context.Context, row R) error {
	return s.mutateThenLoad(ctx, Mutation{Kind: MutUpdate, Record: s.codec.Encode(row)})
}

// Delete removes the row whose key field equals key, then reloads.
func (s *Store[R]) Delete(ctx context.Context, key Value) error {
	return s.mutateThenLoad(ctx, Mutation{Kind: MutDelete, Key: key})
}

// mutateThenLoad applies a write and, on success, reloads the list.
func (s *Store[R]) mutateThenLoad(ctx context.Context, m Mutation) error {
	if err := s.proxy.Mutate(ctx, m); err != nil {
		return err
	}
	_, err := s.Load(ctx)
	return err
}
