#!/bin/bash
# Quick test script to verify benchmark discovery

cd /home/mirai/Documents/Programming/Projects/watt

echo "Testing benchmark discovery..."
echo ""

# Just run discovery without actually running benchmarks
# We'll modify the tool to support this, but for now let's just test with a very short time

timeout 10s ./benchstat/benchstat -v -total-time 5s -benchtime 10ms -count 1 2>&1 | head -50
