#!/bin/bash
set -e

CASE=$1
GATEWAY=$2

if [ -z "$CASE" ] || [ -z "$GATEWAY" ]; then
    echo "Usage: $0 <case> <gateway>"
    exit 1
fi

echo "Running benchmark: Case=$CASE, Gateway=$GATEWAY"

cleanup() {
    echo "Cleaning up..."
    docker compose -f bench/docker-compose.yml down
}
trap cleanup EXIT

# 1. Start Backend and Gateway
echo "Starting backend and gateway $GATEWAY..."
docker compose -f bench/docker-compose.yml up -d --build backend $GATEWAY

# Wait for gateway
sleep 5

# 2. Run Benchmark (wrk)
echo "Running wrk..."
# Load case definition
source "bench/cases/$CASE.sh"

# Ensure results directory exists
mkdir -p bench/results

# Run wrk
wrk $WRK_ARGS http://127.0.0.1:8080/ > "bench/results/${CASE}_${GATEWAY}.txt"

# 3. Parse Results
# In a real script, we'd parse the text output to JSON.
cat "bench/results/${CASE}_${GATEWAY}.txt"

echo "Parsing results..."
python3 bench/runner/parse.py "bench/results/${CASE}_${GATEWAY}.txt" "$GATEWAY" "$CASE"

# 4. Cleanup (handled by trap)

