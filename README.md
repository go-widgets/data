# go-widgets/data

The headless **data spine** of the [go-widgets](https://github.com/go-widgets)
UI ecosystem: a typed record model with validation, a bindable collection with
sort / filter / group / pagination / aggregation, and a **pluggable proxy** so
the *very same* query pipeline runs in-process or against a remote service —
natively and in a browser/wasm build alike.

Pure Go, `CGO=0`, BSD-3-Clause. The core package imports only the standard
library and [`go-widgets/mvvm`](https://github.com/go-widgets/mvvm); nothing here
depends on a GUI.

## Layers

| Piece | What it is |
|-------|-----------|
| `Value` / `Kind` | a comparable typed scalar (string / int / float / bool) |
| `Record` / `Schema` / `Field` / `Rule` | a typed row and its validation |
| `Query` → `Apply` → `View` | the pure query engine: filter, sort, group, page, aggregate |
| `Proxy` | the backend seam: `List` / `Query` / `Mutate` |
| `MemoryProxy` | the in-process reference backend |
| `grpcproxy.Server` / `grpcproxy.Client` | the same contract over gRPC, carried by [`grpc-transports/websocket`](https://github.com/grpc-transports/websocket) |
| `Store[R]` | a typed, bindable collection wired through `mvvm.ObservableList` |

## Why a proxy seam

`Store` talks to a `Proxy`; it never knows whether the data is a `MemoryProxy` in
the same process or a `grpcproxy.Client` reaching a server across a websocket.
Because the query engine (`Apply`) is a pure function of `(records, Query)`, the
client and the server run *the same code* on the same rows — so a `MemoryProxy`
and a `grpcproxy.Client` return **byte-identical** Views.

That equality is asserted directly: `grpcproxy`'s conformance test runs a battery
of sort→filter→group→page→aggregate queries through both proxies and through the
engine, canonicalises every resulting `View`, and requires all three to be
identical byte for byte. The WebSocket transport compiles to `js/wasm`, so the
`grpcproxy.Client` is exactly what a go-widgets wasm app uses to speak this same
service from the browser — no second data path.

## Example

```go
schema := data.Schema{Fields: []data.Field{
    {Name: "id", Kind: data.KindInt},
    {Name: "name", Kind: data.KindString, Rules: []data.Rule{data.Required("name required")}},
    {Name: "salary", Kind: data.KindFloat},
}}
mem, _ := data.NewMemoryProxy(schema, "id",
    data.Record{"id": data.Int(1), "name": data.String("ann"), "salary": data.Float(30)},
)

type person struct{ ID int64; Name string; Salary float64 }
store := data.NewStore(mem, data.Codec[person]{
    Encode: func(p person) data.Record {
        return data.Record{"id": data.Int(p.ID), "name": data.String(p.Name), "salary": data.Float(p.Salary)}
    },
    Decode: func(r data.Record) person {
        return person{ID: r["id"].Int, Name: r["name"].Str, Salary: r["salary"].Float}
    },
})
store.SetQuery(data.Query{Sorts: []data.Sort{{Field: "salary", Desc: true}}, Limit: 20})
store.Load(ctx)                 // runs the query, fills the ObservableList
_ = store.Items()               // bind this to a view — it re-renders on change
```

Swap `mem` for a `grpcproxy.Client` and nothing else changes.

## Status

Core `data` and `grpcproxy` are at 100% statement coverage (the generated
`datapb/` protobuf code excepted); CI builds all six 64-bit Go targets plus
`js/wasm`, macOS and Windows, and runs the race detector. Org-conformance
landing/logo/docs are a follow-up.
