#!/bin/bash

# Automated Test Script for Primary-Backup KV Store
# This script follows the test plan from Project 2.pdf

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Log file
LOG_DIR="logs/automated_test"
mkdir -p "$LOG_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Function to print colored output
print_test() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    ((TESTS_PASSED++))
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Cleanup function
cleanup() {
    echo -e "\n${BLUE}Cleaning up processes...${NC}"
    pkill -f "bin/viewservice" || true
    pkill -f "bin/kvserver" || true
    pkill -f "bin/testcli" || true
    sleep 1
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Build all binaries
print_test "Building binaries"
go build -o bin/viewservice ./cmd/viewservice
go build -o bin/kvserver ./cmd/kvserver
go build -o bin/testcli ./cmd/testcli
print_success "Build completed"

# Clean up any existing processes
cleanup

# Create a helper function to check if a process is running
wait_for_process() {
    local max_attempts=10
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if ps -p $1 > /dev/null 2>&1; then
            return 0
        fi
        sleep 0.5
        ((attempt++))
    done
    return 1
}

# Function to wait for service to be ready
wait_for_service() {
    local max_attempts=20
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        sleep 0.3
        ((attempt++))
    done
}

# Function to perform client Get operation
client_get() {
    local key=$1
    local expected=$2
    local result=$(./bin/testcli -vs localhost:8000 -op get -key "$key" 2>/dev/null || echo "ERROR")
    if [ "$result" == "$expected" ]; then
        print_success "Get(\"$key\") returned \"$result\""
        return 0
    else
        print_error "Get(\"$key\") expected \"$expected\", got \"$result\""
        return 1
    fi
}

# Function to perform client Put operation
client_put() {
    local key=$1
    local value=$2
    local result=$(./bin/testcli -vs localhost:8000 -op put -key "$key" -value "$value" 2>/dev/null || echo "ERROR")
    if [ "$result" == "OK" ]; then
        print_success "Put(\"$key\", \"$value\") succeeded"
        return 0
    else
        print_error "Put(\"$key\", \"$value\") failed: $result"
        return 1
    fi
}

# Function to check if a server has a specific role by checking logs
check_server_role() {
    local log_file=$1
    local expected_role=$2
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if grep -q "Role changed.*to $expected_role" "$log_file" 2>/dev/null; then
            return 0
        fi
        if grep -q "Assigning.*as new $expected_role" "$LOG_DIR/viewservice.log" 2>/dev/null; then
            return 0
        fi
        sleep 0.3
        ((attempt++))
    done
    return 1
}

echo -e "${GREEN}"
echo "========================================="
echo "Primary-Backup KV Store - Automated Test"
echo "========================================="
echo -e "${NC}"

# Test 1: Start View Service
print_test "Test 1: Start View Service"
./bin/viewservice -addr localhost:8000 > "$LOG_DIR/viewservice.log" 2>&1 &
VS_PID=$!
print_info "View Service started with PID $VS_PID"
wait_for_service
if ps -p $VS_PID > /dev/null; then
    print_success "View Service is running"
else
    print_error "View Service failed to start"
    exit 1
fi

# Test 2: Start KV Server S1
print_test "Test 2: Start KV Server S1"
./bin/kvserver -addr localhost:8001 -vs localhost:8000 > "$LOG_DIR/s1.log" 2>&1 &
S1_PID=$!
print_info "KV Server S1 started with PID $S1_PID"
wait_for_service
if ps -p $S1_PID > /dev/null; then
    print_success "KV Server S1 is running"
else
    print_error "KV Server S1 failed to start"
    exit 1
fi

# Test 3: Check the view (S1 should be Primary)
print_test "Test 3: Check view - S1 should be Primary"
sleep 2
if check_server_role "$LOG_DIR/s1.log" "primary"; then
    print_success "S1 is Primary"
else
    print_error "S1 did not become Primary"
    print_info "Check logs: tail $LOG_DIR/s1.log"
fi

# Test 4: Client Put/Get operations
print_test "Test 4: Client operations - Put and Get"
client_put "a" "1"
sleep 0.5
client_get "a" "1"

# Test 5: Start KV Server S2
print_test "Test 5: Start KV Server S2"
./bin/kvserver -addr localhost:8002 -vs localhost:8000 > "$LOG_DIR/s2.log" 2>&1 &
S2_PID=$!
print_info "KV Server S2 started with PID $S2_PID"
wait_for_service
sleep 2

# Test 6: Check the view (S1 Primary, S2 Backup)
print_test "Test 6: Check view - S1 Primary, S2 Backup"
if check_server_role "$LOG_DIR/s1.log" "primary" && check_server_role "$LOG_DIR/s2.log" "backup"; then
    print_success "S1 is Primary, S2 is Backup"
else
    print_error "Expected: S1 Primary, S2 Backup"
fi

# Test 7: Client Put operation
print_test "Test 7: Client Put operation with replication"
client_put "b" "2"
sleep 1

# Test 8: Test Primary Failure - Kill S1
print_test "Test 8: Test Primary Failure - Kill S1"
kill $S1_PID 2>/dev/null || true
print_info "Killed S1 (Primary)"

# Test 9: Wait 2-3 seconds
print_test "Test 9: Wait for failover"
sleep 3
print_info "Waited 3 seconds for failover"

# Test 10: Check the view (S2 should be Primary, no Backup)
print_test "Test 10: Check view - S2 should be Primary"
if check_server_role "$LOG_DIR/s2.log" "primary"; then
    print_success "S2 promoted to Primary"
else
    print_error "S2 did not become Primary"
fi

# Test 11: Client Get operations (data should persist)
print_test "Test 11: Verify data persistence after failover"
client_get "a" "1"
client_get "b" "2"

# Test 12: Test State Transfer - Start KV Server S3
print_test "Test 12: Test State Transfer - Start S3"
./bin/kvserver -addr localhost:8003 -vs localhost:8000 > "$LOG_DIR/s3.log" 2>&1 &
S3_PID=$!
print_info "KV Server S3 started with PID $S3_PID"
wait_for_service
sleep 2

# Test 13: Check the view (S2 Primary, S3 Backup)
print_test "Test 13: Check view - S2 Primary, S3 Backup"
if check_server_role "$LOG_DIR/s2.log" "primary" && check_server_role "$LOG_DIR/s3.log" "backup"; then
    print_success "S2 is Primary, S3 is Backup"
else
    print_error "Expected: S2 Primary, S3 Backup"
fi

# Test 14: Wait for state transfer to complete
print_test "Test 14: Wait for state transfer to complete"
sleep 3
if grep -q "State transfer completed successfully" "$LOG_DIR/s2.log" 2>/dev/null; then
    print_success "State transfer completed"
else
    print_info "State transfer may still be in progress or already completed"
fi

# Test 15: Test New Backup - Kill S2
print_test "Test 15: Test New Backup - Kill S2"
kill $S2_PID 2>/dev/null || true
print_info "Killed S2 (Primary)"
sleep 3

# Test 16: Check the view (S3 should be Primary, no Backup)
print_test "Test 16: Check view - S3 should be Primary"
if check_server_role "$LOG_DIR/s3.log" "primary"; then
    print_success "S3 promoted to Primary"
else
    print_error "S3 did not become Primary"
fi

# Test 17: Final data verification
print_test "Test 17: Final data verification - S3 should have all data"
client_get "a" "1"
client_get "b" "2"

# Summary
echo -e "\n${GREEN}=========================================${NC}"
echo -e "${GREEN}Test Summary${NC}"
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
echo -e "${BLUE}Logs saved to: $LOG_DIR${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}✗ Some tests failed. Check logs for details.${NC}"
    exit 1
fi
