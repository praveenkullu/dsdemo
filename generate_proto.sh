#!/bin/bash

# Generate Go code from proto files

# Create output directories
mkdir -p proto/viewservice
mkdir -p proto/kvserver

# Generate viewservice proto
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/viewservice.proto

# Generate kvserver proto
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/kvserver.proto

echo "Proto files generated successfully!"
