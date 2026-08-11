// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package grpcproxy

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/go-widgets/data"
	"github.com/go-widgets/data/datapb"
	"google.golang.org/grpc"
)

// serveMemory starts a Server over a fresh MemoryProxy seeded with confRows and
// returns a connected Client, cleaning both up via t.Cleanup.
func serveMemory(t *testing.T) (*data.MemoryProxy, *Client) {
	t.Helper()
	mem, err := data.NewMemoryProxy(confSchema(), "id", confRows()...)
	if err != nil {
		t.Fatalf("memory: %v", err)
	}
	url, stop, err := Serve(":0", mem)
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	t.Cleanup(stop)
	client, err := Dial(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return mem, client
}

func TestListParity(t *testing.T) {
	mem, client := serveMemory(t)
	ctx := context.Background()

	local, err := mem.List(ctx)
	if err != nil {
		t.Fatalf("local list: %v", err)
	}
	remote, err := client.List(ctx)
	if err != nil {
		t.Fatalf("remote list: %v", err)
	}
	// List order is the storage order on both sides; compare canonically per row.
	if len(local) != len(remote) {
		t.Fatalf("len local %d != remote %d", len(local), len(remote))
	}
	for i := range local {
		lv := data.Canonical(data.View{Rows: []data.Record{local[i]}, Total: 1})
		rv := data.Canonical(data.View{Rows: []data.Record{remote[i]}, Total: 1})
		if !bytes.Equal(lv, rv) {
			t.Fatalf("row %d differs:\n%s\n%s", i, lv, rv)
		}
	}
}

func TestMutateThroughClientReflectsOnServer(t *testing.T) {
	mem, client := serveMemory(t)
	ctx := context.Background()

	// Insert via the client.
	if err := client.Mutate(ctx, data.Mutation{
		Kind:   data.MutInsert,
		Record: confRow(7, "gwen", "green", 15, true),
	}); err != nil {
		t.Fatalf("client insert: %v", err)
	}
	if list, _ := mem.List(ctx); len(list) != 7 {
		t.Fatalf("server has %d rows after insert, want 7", len(list))
	}

	// Update via the client.
	if err := client.Mutate(ctx, data.Mutation{
		Kind:   data.MutUpdate,
		Record: confRow(7, "gwenn", "green", 16, true),
	}); err != nil {
		t.Fatalf("client update: %v", err)
	}
	v, _ := mem.Query(ctx, data.Query{Filters: []data.Filter{{Field: "id", Op: data.OpEq, Value: data.Int(7)}}})
	if v.Rows[0]["name"] != data.String("gwenn") {
		t.Fatalf("update not reflected: %+v", v.Rows[0])
	}

	// Delete via the client (carries only the key — exercises nil-Record mutation).
	if err := client.Mutate(ctx, data.Mutation{Kind: data.MutDelete, Key: data.Int(7)}); err != nil {
		t.Fatalf("client delete: %v", err)
	}
	if list, _ := mem.List(ctx); len(list) != 6 {
		t.Fatalf("server has %d rows after delete, want 6", len(list))
	}

	// A duplicate insert surfaces the server-side error to the client.
	if err := client.Mutate(ctx, data.Mutation{Kind: data.MutInsert, Record: confRow(1, "dup", "red", 1, true)}); err == nil {
		t.Fatal("expected duplicate-key error through the client")
	}
}

// failProxy makes every operation fail, to cover the Server error branches.
type failProxy struct{ err error }

func (f failProxy) List(context.Context) ([]data.Record, error) { return nil, f.err }
func (f failProxy) Query(context.Context, data.Query) (data.View, error) {
	return data.View{}, f.err
}
func (f failProxy) Mutate(context.Context, data.Mutation) error { return f.err }

func TestServerErrorsPropagate(t *testing.T) {
	url, stop, err := Serve(":0", failProxy{err: errors.New("backend down")})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	defer stop()
	client, err := Dial(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()
	ctx := context.Background()

	if _, err := client.List(ctx); err == nil {
		t.Fatal("List should surface the backend error")
	}
	if _, err := client.Query(ctx, data.Query{}); err == nil {
		t.Fatal("Query should surface the backend error")
	}
	if err := client.Mutate(ctx, data.Mutation{Kind: data.MutInsert, Record: confRow(1, "a", "b", 1, true)}); err == nil {
		t.Fatal("Mutate should surface the backend error")
	}
}

func TestDialErrors(t *testing.T) {
	// Empty URL fails in wstransport.DialOption.
	if _, err := Dial(""); err == nil {
		t.Fatal("Dial(\"\") should fail")
	}
	// grpc.NewClient failure via the seam.
	orig := newGRPCClient
	newGRPCClient = func(string, ...grpc.DialOption) (*grpc.ClientConn, error) {
		return nil, errors.New("newclient boom")
	}
	defer func() { newGRPCClient = orig }()
	if _, err := Dial("ws://127.0.0.1:1/"); err == nil {
		t.Fatal("Dial should fail when grpc.NewClient errors")
	}
}

// TestServeBadAddress covers the Serve listen-error branch.
func TestServeBadAddress(t *testing.T) {
	if _, _, err := Serve("256.256.256.256:99999", failProxy{}); err == nil {
		t.Fatal("Serve should fail to bind a bad address")
	}
}

// TestConversionNilBranches covers the nil-guard branches in conv.go that the
// happy-path round trips do not reach on their own.
func TestConversionNilBranches(t *testing.T) {
	if v := valueFromPB(nil); v != (data.Value{}) {
		t.Fatalf("valueFromPB(nil) = %+v", v)
	}
	if r := recordFromPB(nil); r != nil {
		t.Fatalf("recordFromPB(nil) = %+v", r)
	}
	if m := aggsFromPB(nil); m != nil {
		t.Fatal("aggsFromPB(nil) should be nil")
	}
	if m := aggsToPB(nil); m != nil {
		t.Fatal("aggsToPB(nil) should be nil")
	}
	// A view carrying an empty (non-nil) aggregate map still round-trips to nil,
	// and an ungrouped empty view stays ungrouped.
	pb := viewToPB(data.View{Rows: []data.Record{}, Total: 0})
	if pb.GetGrouped() {
		t.Fatal("ungrouped view marked grouped")
	}
	if got := viewFromPB(&datapb.View{Grouped: false}); got.Groups != nil {
		t.Fatalf("ungrouped decode has groups: %+v", got)
	}
}
