.PHONY: build clean test test-auto run-vs run-s1 run-s2 run-s3 run-client help

# Default target
all: build

# Build all binaries
build:
	@echo "Building all binaries..."
	@mkdir -p bin
	@mkdir -p logs
	@go build -o bin/viewservice ./cmd/viewservice
	@go build -o bin/kvserver ./cmd/kvserver
	@go build -o bin/testclient ./cmd/testclient
	@go build -o bin/testcli ./cmd/testcli
	@echo "✓ Build completed"

# Clean build artifacts and logs
clean:
	@echo "Cleaning up..."
	@rm -rf bin logs
	@pkill -f "viewservice" || true
	@pkill -f "kvserver" || true
	@pkill -f "testclient" || true
	@echo "✓ Cleanup completed"

# Run view service
run-vs: build
	@mkdir -p logs
	@echo "Starting View Service on localhost:8000..."
	@./bin/viewservice -addr localhost:8000

# Run KV server 1
run-s1: build
	@mkdir -p logs
	@echo "Starting KV Server S1 on localhost:8001..."
	@./bin/kvserver -addr localhost:8001 -vs localhost:8000

# Run KV server 2
run-s2: build
	@mkdir -p logs
	@echo "Starting KV Server S2 on localhost:8002..."
	@./bin/kvserver -addr localhost:8002 -vs localhost:8000

# Run KV server 3
run-s3: build
	@mkdir -p logs
	@echo "Starting KV Server S3 on localhost:8003..."
	@./bin/kvserver -addr localhost:8003 -vs localhost:8000

# Run test client
run-client: build
	@echo "Starting Test Client..."
	@./bin/testclient -vs localhost:8000

# Run comprehensive test
test: build
	@mkdir -p logs
	@chmod +x test.sh
	@./test.sh

# Run automated test suite
test-auto: build
	@mkdir -p logs
	@chmod +x automated_test.sh
	@./automated_test.sh

# Download dependencies
deps:
	@echo "Downloading Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

# Help
help:
	@echo "Primary-Backup KV Store - Makefile commands:"
	@echo ""
	@echo "  make build       - Build all binaries"
	@echo "  make clean       - Clean build artifacts and kill processes"
	@echo "  make run-vs      - Run View Service"
	@echo "  make run-s1      - Run KV Server S1"
	@echo "  make run-s2      - Run KV Server S2"
	@echo "  make run-s3      - Run KV Server S3"
	@echo "  make run-client  - Run test client"
	@echo "  make test        - Run manual test suite"
	@echo "  make test-auto   - Run automated test suite (recommended)"
	@echo "  make deps        - Download and update dependencies"
	@echo "  make help        - Show this help message"
	@echo ""
	@echo "Quick start:"
	@echo "  Terminal 1: make run-vs"
	@echo "  Terminal 2: make run-s1"
	@echo "  Terminal 3: make run-s2"
	@echo "  Terminal 4: make run-client"
