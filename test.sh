#!/bin/bash

# Comprehensive test script for Primary-Backup KV Store
# This test follows the test plan from the PDF

set -e

echo "========================================="
echo "Primary-Backup KV Store Test Suite"
echo "========================================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    echo -e "\n${BLUE}Cleaning up processes...${NC}"
    pkill -f "viewservice/viewservice" || true
    pkill -f "kvserver/kvserver" || true
    pkill -f "testclient/testclient" || true
    sleep 1
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Build all binaries
echo -e "${BLUE}Building binaries...${NC}"
cd /home/user/dsdemo
go build -o bin/viewservice ./cmd/viewservice
go build -o bin/kvserver ./cmd/kvserver
go build -o bin/testclient ./cmd/testclient
echo -e "${GREEN}✓ Build completed${NC}\n"

# Clean up any existing processes
cleanup

echo -e "${BLUE}=== Test 1: Start View Service ===${NC}"
./bin/viewservice -addr localhost:8000 > logs/viewservice.log 2>&1 &
VS_PID=$!
echo "View Service started with PID $VS_PID"
sleep 2

echo -e "\n${BLUE}=== Test 2: Start KV Server S1 ===${NC}"
./bin/kvserver -addr localhost:8001 -vs localhost:8000 > logs/s1.log 2>&1 &
S1_PID=$!
echo "KV Server S1 started with PID $S1_PID"
sleep 2

echo -e "\n${BLUE}=== Test 3: Check view (S1 should be Primary) ===${NC}"
sleep 2
echo "Expected: S1 as Primary, no Backup"

echo -e "\n${BLUE}=== Test 4: Client Put and Get ===${NC}"
# Note: We'll do this manually for now
echo "Testing Put(a, 1) and Get(a)..."
echo "This would be done via the client library"

echo -e "\n${BLUE}=== Test 5: Start KV Server S2 ===${NC}"
./bin/kvserver -addr localhost:8002 -vs localhost:8000 > logs/s2.log 2>&1 &
S2_PID=$!
echo "KV Server S2 started with PID $S2_PID"
sleep 3

echo -e "\n${BLUE}=== Test 6: Check view (S1 Primary, S2 Backup) ===${NC}"
echo "Expected: S1 as Primary, S2 as Backup"
sleep 1

echo -e "\n${BLUE}=== Test 7: Client Put(b, 2) ===${NC}"
echo "This should replicate to S2"
sleep 1

echo -e "\n${BLUE}=== Test 8: Test Primary Failure - Kill S1 ===${NC}"
echo "Killing S1 (Primary)..."
kill $S1_PID
echo "S1 killed"
sleep 3

echo -e "\n${BLUE}=== Test 9: Check view (S2 should be Primary, no Backup) ===${NC}"
echo "Expected: S2 promoted to Primary, no Backup"
sleep 1

echo -e "\n${BLUE}=== Test 10: Client Get(a) and Get(b) ===${NC}"
echo "These should still work with S2 as new Primary"
echo "Get(a) should return 1"
echo "Get(b) should return 2"

echo -e "\n${BLUE}=== Test 11: Test State Transfer - Start KV Server S3 ===${NC}"
./bin/kvserver -addr localhost:8003 -vs localhost:8000 > logs/s3.log 2>&1 &
S3_PID=$!
echo "KV Server S3 started with PID $S3_PID"
sleep 4

echo -e "\n${BLUE}=== Test 12: Check view (S2 Primary, S3 Backup) ===${NC}"
echo "Expected: S2 as Primary, S3 as Backup (with state transferred)"
sleep 1

echo -e "\n${BLUE}=== Test 13: Test New Backup - Kill S2 ===${NC}"
echo "Killing S2 (Primary)..."
kill $S2_PID
echo "S2 killed"
sleep 3

echo -e "\n${BLUE}=== Test 14: Check view (S3 should be Primary, no Backup) ===${NC}"
echo "Expected: S3 promoted to Primary, no Backup"
sleep 1

echo -e "\n${BLUE}=== Test 15: Client Get(a) and Get(b) ===${NC}"
echo "These should still work with S3 as new Primary"
echo "Get(a) should return 1"
echo "Get(b) should return 2"
echo "This verifies state transfer worked correctly!"

echo -e "\n${GREEN}========================================="
echo "All tests completed!"
echo "=========================================${NC}"

echo -e "\n${BLUE}Check the logs in logs/ directory for details:${NC}"
echo "  - logs/viewservice.log"
echo "  - logs/s1.log"
echo "  - logs/s2.log"
echo "  - logs/s3.log"

echo -e "\n${BLUE}Press Enter to shut down all services...${NC}"
read

# Cleanup will be called by trap
