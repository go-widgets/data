// Copyright (c) 2026 the go-widgets/data authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

//go:build !js

package grpcproxy

import (
	"context"
	"fmt"

	"github.com/go-widgets/data"
	"github.com/go-widgets/data/datapb"
	wstransport "github.com/grpc-transports/websocket"
	"google.golang.org/grpc"
)

// Server adapts any data.Proxy to the DataService gRPC contract: each RPC simply
// delegates to the wrapped proxy (a MemoryProxy in practice), so the query
// pipeline that produces a View runs server-side and the client receives exactly
// that View. It is native-only — a browser is the client, never the server.
type Server struct {
	datapb.UnimplementedDataServiceServer
	proxy data.Proxy
}

// NewServer wraps proxy as a DataService implementation.
func NewServer(proxy data.Proxy) *Server { return &Server{proxy: proxy} }

// List returns every record.
func (s *Server) List(ctx context.Context, _ *datapb.ListRequest) (*datapb.ListResponse, error) {
	rows, err := s.proxy.List(ctx)
	if err != nil {
		return nil, err
	}
	return &datapb.ListResponse{Records: recordsToPB(rows)}, nil
}

// Query applies the query on the proxy and returns the resulting View.
func (s *Server) Query(ctx context.Context, q *datapb.Query) (*datapb.View, error) {
	view, err := s.proxy.Query(ctx, queryFromPB(q))
	if err != nil {
		return nil, err
	}
	return viewToPB(view), nil
}

// Mutate applies one write.
func (s *Server) Mutate(ctx context.Context, m *datapb.Mutation) (*datapb.MutateResponse, error) {
	if err := s.proxy.Mutate(ctx, mutationFromPB(m)); err != nil {
		return nil, err
	}
	return &datapb.MutateResponse{}, nil
}

// RegisterServer registers a DataService backed by proxy onto an existing
// grpc.Server — use it when mounting the service alongside others.
func RegisterServer(gs *grpc.Server, proxy data.Proxy) {
	datapb.RegisterDataServiceServer(gs, NewServer(proxy))
}

// Serve starts a DataService for proxy over a WebSocket-carried gRPC listener
// bound to addr (use ":0" for an ephemeral port). It returns the ws:// URL a
// Client dials and a stop function that shuts the server down. The WebSocket
// carrier is what lets a browser/wasm Client speak this same gRPC service.
func Serve(addr string, proxy data.Proxy) (url string, stop func(), err error) {
	lis, err := wstransport.ListenWebSocket(addr, wstransport.ServerConfig{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return "", nil, err
	}
	gs := grpc.NewServer()
	RegisterServer(gs, proxy)
	go func() { _ = gs.Serve(lis) }()
	url = fmt.Sprintf("ws://%s/", lis.Addr().String())
	return url, func() { gs.Stop() }, nil
}
