#!/usr/bin/env bash
set -euo pipefail

# Generate synthetic telemetry load for FusionGuard
# Usage: ./loadgen.sh [options]

INGESTOR_URL="${INGESTOR_URL:-http://localhost:8081}"
RATE_HZ="${RATE_HZ:-1000}"
DURATION_SEC="${DURATION_SEC:-60}"
SHOT_ID="${SHOT_ID:-loadgen_$(date +%s)}"
N_THREADS="${N_THREADS:-1}"

usage() {
    cat <<EOF
Usage: $0 [options]

Options:
    --ingestor-url URL    Ingestor HTTP endpoint (default: http://localhost:8081)
    --rate-hz RATE        Generation rate in Hz per thread (default: 1000)
    --duration-sec SEC    Duration in seconds (default: 60)
    --shot-id ID          Shot ID prefix (default: loadgen_TIMESTAMP)
    --threads N           Number of parallel threads (default: 1)
    -h, --help           Show this help

Example:
    $0 --rate-hz 5000 --duration-sec 120 --threads 4
EOF
    exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --ingestor-url)
            INGESTOR_URL="$2"
            shift 2
            ;;
        --rate-hz)
            RATE_HZ="$2"
            shift 2
            ;;
        --duration-sec)
            DURATION_SEC="$2"
            shift 2
            ;;
        --shot-id)
            SHOT_ID="$2"
            shift 2
            ;;
        --threads)
            N_THREADS="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Check if Python is available
if ! command -v python3 &> /dev/null; then
    echo "Error: python3 is required"
    exit 1
fi

echo "Generating load:"
echo "  Ingestor: $INGESTOR_URL"
echo "  Rate: ${RATE_HZ}Hz per thread"
echo "  Duration: ${DURATION_SEC}s"
echo "  Threads: $N_THREADS"
echo "  Shot ID prefix: $SHOT_ID"
echo ""

# Function to generate and send telemetry in a thread
generate_thread() {
    local thread_id=$1
    local shot_id="${SHOT_ID}_t${thread_id}"
    local rate_hz=$RATE_HZ
    local duration_sec=$DURATION_SEC
    local ingestor_url=$INGESTOR_URL
    
    python3 <<EOF
import json
import random
import sys
import time
import math

shot_id = "$shot_id"
rate_hz = $rate_hz
duration_sec = $duration_sec
ingestor_url = "$ingestor_url"
thread_id = $thread_id

# Calculate timing
delay = 1.0 / rate_hz
n_samples = int(rate_hz * duration_sec)
start_time_ns = int(time.time() * 1_000_000_000)

# Channels to generate
channels = ["ip", "ne", "dwdt", "prad", "h_alpha"]

# Base values
base_values = {
    "ip": 1.0,
    "ne": 0.5,
    "dwdt": 0.0,
    "prad": 0.3,
    "h_alpha": 0.1,
}

sent = 0
errors = 0
last_report = time.time()

for i in range(n_samples):
    # Calculate timestamp
    elapsed_ns = int(i * delay * 1_000_000_000)
    ts_unix_ns = start_time_ns + elapsed_ns
    
    # Generate channel values with some variation
    channel_samples = []
    for ch_name, base_val in base_values.items():
        # Add sinusoidal variation and noise
        t = i * delay
        variation = 0.1 * math.sin(2 * math.pi * t / 2.0)  # 2 second period
        noise = 0.05 * random.gauss(0, 1)
        value = base_val + variation + noise
        
        # Ensure non-negative for physical quantities
        value = max(0.0, value)
        
        channel_samples.append({
            "name": ch_name,
            "value": value,
            "quality": "good"
        })
    
    # Build telemetry point
    point = {
        "shot_id": shot_id,
        "ts_unix_ns": ts_unix_ns,
        "channels": channel_samples
    }
    
    # Send point
    import urllib.request
    import urllib.error
    
    json_data = json.dumps(point).encode('utf-8')
    req = urllib.request.Request(
        f"{ingestor_url}/ingest",
        data=json_data,
        headers={"Content-Type": "application/json"},
        method="POST"
    )
    
    try:
        with urllib.request.urlopen(req, timeout=1.0) as response:
            if response.status != 200:
                errors += 1
            else:
                sent += 1
    except Exception as e:
        errors += 1
        if errors <= 5:  # Only print first few errors
            print(f"Thread {thread_id} error: {e}", file=sys.stderr)
    
    # Rate limiting
    next_time = start_time_ns / 1_000_000_000 + (i + 1) * delay
    sleep_time = next_time - time.time()
    if sleep_time > 0:
        time.sleep(sleep_time)
    
    # Progress report
    if time.time() - last_report >= 5.0:
        print(f"Thread {thread_id}: sent={sent}, errors={errors}", file=sys.stderr)
        last_report = time.time()

print(f"Thread {thread_id} completed: sent={sent}, errors={errors}", file=sys.stderr)
EOF
}

# Start threads
pids=()
for i in $(seq 1 $N_THREADS); do
    generate_thread $i &
    pids+=($!)
done

# Wait for all threads
total_sent=0
total_errors=0

for pid in "${pids[@]}"; do
    wait $pid
done

echo ""
echo "Load generation completed!"
echo "Total threads: $N_THREADS"
echo "Total samples: $((N_THREADS * RATE_HZ * DURATION_SEC))"
