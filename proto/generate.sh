#!/bin/sh
# Regenerate datapb/*.pb.go from proto/data.proto. Requires protoc plus
# protoc-gen-go (v1.36.x) and protoc-gen-go-grpc (v1.5.x) on PATH:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
set -e
cd "$(dirname "$0")/.."
protoc \
  --go_out=. --go_opt=module=github.com/go-widgets/data \
  --go-grpc_out=. --go-grpc_opt=module=github.com/go-widgets/data \
  proto/data.proto
echo "regenerated datapb/"
