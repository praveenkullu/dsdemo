#!/bin/bash

# Simple demo script for Primary-Backup KV Store
# This script demonstrates basic functionality

echo "========================================="
echo "Primary-Backup KV Store - Quick Demo"
echo "========================================="

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    echo -e "\n${BLUE}Cleaning up processes...${NC}"
    pkill -f "bin/viewservice" || true
    pkill -f "bin/kvserver" || true
    sleep 1
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Clean up any existing processes
cleanup

# Build if needed
if [ ! -f "bin/viewservice" ] || [ ! -f "bin/kvserver" ] || [ ! -f "bin/testclient" ]; then
    echo -e "${BLUE}Building binaries...${NC}"
    make build
fi

echo -e "\n${GREEN}Starting Primary-Backup KV Store Demo${NC}\n"

echo -e "${BLUE}Step 1: Starting View Service${NC}"
./bin/viewservice -addr localhost:8000 > logs/demo_vs.log 2>&1 &
VS_PID=$!
echo "  View Service started (PID: $VS_PID)"
sleep 2

echo -e "\n${BLUE}Step 2: Starting KV Server S1${NC}"
./bin/kvserver -addr localhost:8001 -vs localhost:8000 > logs/demo_s1.log 2>&1 &
S1_PID=$!
echo "  KV Server S1 started (PID: $S1_PID)"
echo "  S1 will become Primary"
sleep 3

echo -e "\n${BLUE}Step 3: Starting KV Server S2${NC}"
./bin/kvserver -addr localhost:8002 -vs localhost:8000 > logs/demo_s2.log 2>&1 &
S2_PID=$!
echo "  KV Server S2 started (PID: $S2_PID)"
echo "  S2 will become Backup"
sleep 3

echo -e "\n${GREEN}System is now running!${NC}"
echo -e "${YELLOW}Current configuration:${NC}"
echo "  Primary: S1 (localhost:8001)"
echo "  Backup:  S2 (localhost:8002)"

echo -e "\n${BLUE}Step 4: Running test client${NC}"
echo "  The client will perform continuous Put/Get operations"
echo "  You can kill S1 (Primary) to see automatic failover to S2"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop the demo${NC}\n"

sleep 2

# Run the test client in foreground
./bin/testclient -vs localhost:8000

# Cleanup will be called by trap
