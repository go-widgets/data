// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package grpcproxy

import (
	"context"

	"github.com/go-widgets/data"
	"github.com/go-widgets/data/datapb"
	wstransport "github.com/grpc-transports/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a data.Proxy backed by a remote DataService reached over a
// WebSocket-carried gRPC connection. It satisfies the same interface a
// MemoryProxy does, so a Store cannot tell whether its data is local or remote —
// and because the WebSocket transport compiles to js/wasm, this exact Client
// runs in a browser talking to the native Server.
type Client struct {
	cc   *grpc.ClientConn
	stub datapb.DataServiceClient
}

// compile-time check that Client is a data.Proxy.
var _ data.Proxy = (*Client)(nil)

// newGRPCClient is the grpc.NewClient seam, indirected so a test can force its
// error branch (a passthrough target never fails in practice).
var newGRPCClient = grpc.NewClient

// Dial connects to a DataService at the given ws:// (or wss://) URL. Close the
// returned Client when done.
func Dial(url string) (*Client, error) {
	dialOpt, err := wstransport.DialOption(url, wstransport.ClientConfig{})
	if err != nil {
		return nil, err
	}
	cc, err := newGRPCClient("passthrough:///"+url,
		dialOpt,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &Client{cc: cc, stub: datapb.NewDataServiceClient(cc)}, nil
}

// Close releases the underlying connection.
func (c *Client) Close() error { return c.cc.Close() }

// List returns every record from the remote service.
func (c *Client) List(ctx context.Context) ([]data.Record, error) {
	resp, err := c.stub.List(ctx, &datapb.ListRequest{})
	if err != nil {
		return nil, err
	}
	return recordsFromPB(resp.GetRecords()), nil
}

// Query runs the query remotely and returns the View the server computed — the
// same View a MemoryProxy would have produced for the same rows.
func (c *Client) Query(ctx context.Context, q data.Query) (data.View, error) {
	resp, err := c.stub.Query(ctx, queryToPB(q))
	if err != nil {
		return data.View{}, err
	}
	return viewFromPB(resp), nil
}

// Mutate applies one write remotely.
func (c *Client) Mutate(ctx context.Context, m data.Mutation) error {
	_, err := c.stub.Mutate(ctx, mutationToPB(m))
	return err
}
