#!/bin/bash

# Generate Go code from proto files using gRPC

# Add Go bin to PATH (for protoc-gen-go and protoc-gen-go-grpc)
export PATH=$PATH:$(go env GOPATH)/bin

# Generate proto files
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/viewservice.proto proto/kvserver.proto

echo "Proto files generated successfully!"
echo "Generated files:"
ls -lh proto/*.pb.go
