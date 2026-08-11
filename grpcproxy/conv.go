// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package grpcproxy carries the data spine over gRPC: a Server that serves any
// data.Proxy, and a Client (itself a data.Proxy) that reaches it. The transport
// is grpc-transports/websocket, so the exact same query pipeline runs whether a
// Store talks to a MemoryProxy in-process or to this Client from a browser/wasm
// build — and the Views come back byte-identical.
package grpcproxy

import (
	"github.com/go-widgets/data"
	"github.com/go-widgets/data/datapb"
)

// The data.* scalar enums are declared in the same order as their protobuf
// counterparts, so each converts by a plain numeric cast. These helpers keep the
// casts in one place and self-documenting.

func valueToPB(v data.Value) *datapb.Value {
	return &datapb.Value{
		Kind:  datapb.Kind(v.Kind),
		Str:   v.Str,
		Int:   v.Int,
		Float: v.Float,
		Bool:  v.Bool,
	}
}

func valueFromPB(v *datapb.Value) data.Value {
	if v == nil {
		return data.Value{}
	}
	return data.Value{
		Kind:  data.Kind(v.GetKind()),
		Str:   v.GetStr(),
		Int:   v.GetInt(),
		Float: v.GetFloat(),
		Bool:  v.GetBool(),
	}
}

func recordToPB(r data.Record) *datapb.Record {
	fields := make(map[string]*datapb.Value, len(r))
	for k, v := range r {
		fields[k] = valueToPB(v)
	}
	return &datapb.Record{Fields: fields}
}

func recordFromPB(r *datapb.Record) data.Record {
	if r == nil {
		return nil
	}
	out := make(data.Record, len(r.GetFields()))
	for k, v := range r.GetFields() {
		out[k] = valueFromPB(v)
	}
	return out
}

func recordsToPB(rows []data.Record) []*datapb.Record {
	out := make([]*datapb.Record, len(rows))
	for i, r := range rows {
		out[i] = recordToPB(r)
	}
	return out
}

func recordsFromPB(rows []*datapb.Record) []data.Record {
	out := make([]data.Record, len(rows))
	for i, r := range rows {
		out[i] = recordFromPB(r)
	}
	return out
}

func aggsToPB(m map[string]data.Value) map[string]*datapb.Value {
	if m == nil {
		return nil
	}
	out := make(map[string]*datapb.Value, len(m))
	for k, v := range m {
		out[k] = valueToPB(v)
	}
	return out
}

func aggsFromPB(m map[string]*datapb.Value) map[string]data.Value {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]data.Value, len(m))
	for k, v := range m {
		out[k] = valueFromPB(v)
	}
	return out
}

func queryToPB(q data.Query) *datapb.Query {
	filters := make([]*datapb.Filter, len(q.Filters))
	for i, f := range q.Filters {
		filters[i] = &datapb.Filter{Field: f.Field, Op: datapb.FilterOp(f.Op), Value: valueToPB(f.Value)}
	}
	sorts := make([]*datapb.Sort, len(q.Sorts))
	for i, s := range q.Sorts {
		sorts[i] = &datapb.Sort{Field: s.Field, Desc: s.Desc}
	}
	aggs := make([]*datapb.Agg, len(q.Aggs))
	for i, a := range q.Aggs {
		aggs[i] = &datapb.Agg{Field: a.Field, Func: datapb.AggFunc(a.Func)}
	}
	return &datapb.Query{
		Filters: filters, Sorts: sorts, GroupBy: q.GroupBy,
		Offset: int32(q.Offset), Limit: int32(q.Limit), Aggs: aggs,
	}
}

func queryFromPB(q *datapb.Query) data.Query {
	filters := make([]data.Filter, len(q.GetFilters()))
	for i, f := range q.GetFilters() {
		filters[i] = data.Filter{Field: f.GetField(), Op: data.FilterOp(f.GetOp()), Value: valueFromPB(f.GetValue())}
	}
	sorts := make([]data.Sort, len(q.GetSorts()))
	for i, s := range q.GetSorts() {
		sorts[i] = data.Sort{Field: s.GetField(), Desc: s.GetDesc()}
	}
	aggs := make([]data.Agg, len(q.GetAggs()))
	for i, a := range q.GetAggs() {
		aggs[i] = data.Agg{Field: a.GetField(), Func: data.AggFunc(a.GetFunc())}
	}
	return data.Query{
		Filters: filters, Sorts: sorts, GroupBy: q.GetGroupBy(),
		Offset: int(q.GetOffset()), Limit: int(q.GetLimit()), Aggs: aggs,
	}
}

func viewToPB(v data.View) *datapb.View {
	out := &datapb.View{
		Total:      int32(v.Total),
		Aggregates: aggsToPB(v.Aggregates),
		Grouped:    v.Groups != nil,
	}
	if v.Groups != nil {
		out.Groups = make([]*datapb.Group, len(v.Groups))
		for i, g := range v.Groups {
			out.Groups[i] = &datapb.Group{
				Key:        valueToPB(g.Key),
				Rows:       recordsToPB(g.Rows),
				Aggregates: aggsToPB(g.Aggregates),
			}
		}
		return out
	}
	out.Rows = recordsToPB(v.Rows)
	return out
}

func viewFromPB(v *datapb.View) data.View {
	out := data.View{
		Total:      int(v.GetTotal()),
		Aggregates: aggsFromPB(v.GetAggregates()),
	}
	// The grouped flag is what preserves the nil-Rows / non-nil-Groups shape a
	// grouped View has, which repeated fields alone cannot express.
	if v.GetGrouped() {
		groups := make([]data.Group, len(v.GetGroups()))
		for i, g := range v.GetGroups() {
			groups[i] = data.Group{
				Key:        valueFromPB(g.GetKey()),
				Rows:       recordsFromPB(g.GetRows()),
				Aggregates: aggsFromPB(g.GetAggregates()),
			}
		}
		out.Groups = groups
		return out
	}
	out.Rows = recordsFromPB(v.GetRows())
	return out
}

func mutationToPB(m data.Mutation) *datapb.Mutation {
	out := &datapb.Mutation{Kind: datapb.MutationKind(m.Kind), Key: valueToPB(m.Key)}
	if m.Record != nil {
		out.Record = recordToPB(m.Record)
	}
	return out
}

func mutationFromPB(m *datapb.Mutation) data.Mutation {
	return data.Mutation{
		Kind:   data.MutationKind(m.GetKind()),
		Record: recordFromPB(m.GetRecord()),
		Key:    valueFromPB(m.GetKey()),
	}
}
