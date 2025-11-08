# Primary-Backup Key-Value Store

A fault-tolerant distributed key-value store implementation using the Primary-Backup replication scheme in Go with gRPC. This project demonstrates fundamental distributed systems concepts including:

- Primary-Backup replication
- Failure detection via heartbeating
- Automatic failover
- Synchronous replication
- State transfer
- gRPC-based communication

## System Architecture

The system consists of three main components:

### 1. View Service
A central server that maintains the "view" of the system:
- Decides who is the Primary and who is the Backup
- Detects server failures via heartbeating (Ping RPCs every 0.5s)
- Manages the failover process by promoting the backup
- Acts as a single source of truth for system configuration

### 2. KV Servers (Primary & Backup)
Servers that store the actual key-value data:
- **Primary**: Accepts Get/Put requests from clients
- **Backup**: Maintains a perfect replica of Primary's data
- Both servers periodically ping the View Service to announce they are alive
- Supports synchronous replication (Primary waits for Backup ACK)
- Implements state transfer when a new Backup joins

### 3. Client
A library that provides a simple interface to the KV service:
- Exposes `Get(key)` and `Put(key, value)` functions
- Automatically discovers the current Primary via View Service
- Handles failover transparently by retrying with new Primary

## Project Structure

```
.
├── viewservice/          # View Service implementation
│   └── server.go        # View Service gRPC server logic
├── kvserver/            # KV Server implementation
│   └── server.go        # KV Server gRPC logic with Primary/Backup roles
├── client/              # Client library
│   └── client.go        # gRPC client with automatic failover
├── cmd/                 # Executable programs
│   ├── viewservice/     # View Service binary
│   ├── kvserver/        # KV Server binary
│   └── testclient/      # Test client binary
├── proto/               # Protocol Buffer definitions and generated code
│   ├── viewservice.proto    # View Service protobuf definitions
│   ├── kvserver.proto       # KV Server protobuf definitions
│   ├── *.pb.go             # Generated protobuf code
│   └── *_grpc.pb.go        # Generated gRPC code
├── Makefile            # Build automation
├── test.sh             # Comprehensive test script
├── generate_proto.sh   # Script to regenerate protobuf code
└── go.mod              # Go module definition
```

## Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (protoc) - for regenerating proto files (optional)
- Make (optional, for using Makefile commands)

### Installing protoc (optional - only needed to regenerate proto files)

**Ubuntu/Debian:**
```bash
sudo apt-get install protobuf-compiler
```

**macOS:**
```bash
brew install protobuf
```

**Install Go plugins:**
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Building

### Using Makefile (recommended)

```bash
make build
```

### Manual build

```bash
mkdir -p bin logs
go build -o bin/viewservice ./cmd/viewservice
go build -o bin/kvserver ./cmd/kvserver
go build -o bin/testclient ./cmd/testclient
```

## Running the System

### Quick Start (4 terminals)

**Terminal 1 - View Service:**
```bash
make run-vs
# or
./bin/viewservice -addr localhost:8000
```

**Terminal 2 - KV Server S1:**
```bash
make run-s1
# or
./bin/kvserver -addr localhost:8001 -vs localhost:8000
```

**Terminal 3 - KV Server S2:**
```bash
make run-s2
# or
./bin/kvserver -addr localhost:8002 -vs localhost:8000
```

**Terminal 4 - Test Client:**
```bash
make run-client
# or
./bin/testclient -vs localhost:8000
```

### Command Line Options

**View Service:**
```bash
./bin/viewservice -addr <host:port>
```
- `-addr`: Address to listen on (default: localhost:8000)

**KV Server:**
```bash
./bin/kvserver -addr <host:port> -vs <viewservice_host:port>
```
- `-addr`: Address to listen on (default: localhost:8001)
- `-vs`: View Service address (default: localhost:8000)

**Test Client:**
```bash
./bin/testclient -vs <viewservice_host:port>
```
- `-vs`: View Service address (default: localhost:8000)

## Testing

### Run the comprehensive test suite

```bash
make test
```

This test script follows the test plan from the lab requirements:
1. Starts View Service
2. Starts KV Server S1 (becomes Primary)
3. Performs Put/Get operations
4. Starts KV Server S2 (becomes Backup)
5. Tests Primary failure (kills S1, S2 should be promoted)
6. Tests state transfer (starts S3, should receive full state from S2)
7. Tests new Backup promotion (kills S2, S3 should be promoted)
8. Verifies data integrity throughout all failures

### Manual Testing Scenarios

#### Test 1: Basic Operations
```bash
# Start View Service and two KV servers
# In client, perform:
Put("a", "1")
Get("a")  # Should return "1"
Put("b", "2")
Get("b")  # Should return "2"
```

#### Test 2: Primary Failure
```bash
# With S1 as Primary and S2 as Backup
# Kill S1
# S2 should be promoted to Primary
Get("a")  # Should still return "1"
Get("b")  # Should still return "2"
```

#### Test 3: State Transfer
```bash
# With S2 as Primary (after S1 failed)
# Start S3
# S3 should become Backup and receive full state
# Kill S2
# S3 should be promoted to Primary
Get("a")  # Should still return "1"
Get("b")  # Should still return "2"
```

## Design Details

### View Service Implementation

The View Service tracks server health through periodic pings:
- Servers must ping every 0.5 seconds
- Servers are declared dead after 1.5 seconds without a ping
- View number increments on every view change
- Only promotes Backup to Primary after Primary has acknowledged current view

### KV Server Implementation

#### Primary Role:
- Accepts Get/Put requests from clients
- On Put: forwards update to Backup, waits for ACK, then updates local state
- Initiates state transfer when a new Backup joins
- Queues Put requests during state transfer

#### Backup Role:
- Rejects all client Get/Put requests
- Accepts ForwardUpdate RPCs from Primary
- Accepts SyncState RPC for state transfer
- Ready to be promoted to Primary at any time

### Client Implementation

The client provides transparent failover:
1. Calls GetView() to find current Primary
2. Sends Get/Put to Primary
3. If RPC fails or gets "not primary" error:
   - Calls GetView() again
   - Retries with new Primary
   - Loops until success

### State Transfer Protocol

When a new Backup joins:
1. Primary detects new Backup in view
2. Primary sets `syncing = true`
3. Primary creates a snapshot of all data
4. Primary sends entire snapshot to Backup via SyncState RPC
5. Backup overwrites its local state
6. Primary sets `syncing = false`
7. Primary processes any queued Put requests

## Implementation Notes

- **RPC Framework**: Uses gRPC with Protocol Buffers for efficient, type-safe communication
- **Synchronous Replication**: Primary waits for Backup ACK before responding to client
- **Consistency**: Ensures Backup is never behind Primary
- **Fault Tolerance**: Handles single server failures (Primary or Backup)
- **No Network Partitions**: Assumes reliable network (as per requirements)
- **Single Point of Failure**: View Service is a SPOF (acceptable for this lab)
- **gRPC Features**: Context-based timeouts, connection pooling, and structured error handling

## Known Limitations

1. View Service is a single point of failure
2. Does not handle network partitions
3. Does not handle split-brain scenarios
4. No support for multiple simultaneous failures
5. State transfer blocks new writes (queued until complete)

## Troubleshooting

### Servers not connecting to View Service
- Ensure View Service is running first
- Check that addresses and ports match
- Look for "Connected to view service" in server logs

### Client operations failing
- Verify at least one KV server is running
- Wait 2-3 seconds for servers to become Primary
- Check View Service logs to see current Primary/Backup

### State not preserved after failover
- Ensure Backup was running before Primary failed
- Check that state transfer completed (look for "State transfer completed" in logs)
- Verify View Service promoted Backup to Primary

## Logs

All logs are stored in the `logs/` directory:
- `viewservice.log` - View Service logs
- `s1.log` - KV Server S1 logs
- `s2.log` - KV Server S2 logs
- `s3.log` - KV Server S3 logs

## Cleanup

```bash
make clean
```

This will:
- Remove all built binaries
- Remove all log files
- Kill any running processes

## References

This implementation follows the requirements from:
- **Course**: MSC5703/MCS4993: Intro to Distributed Computing Fall 2025
- **Lab**: Lab 2: Primary-Backup
- **Concepts**: Primary-Backup replication, RPC, fault tolerance, state transfer

## License

Educational project for distributed systems course.
