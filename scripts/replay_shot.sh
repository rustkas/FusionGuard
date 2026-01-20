#!/usr/bin/env bash
set -euo pipefail

# Replay telemetry data from a file through the FusionGuard ingestor
# Usage: ./replay_shot.sh <data_file> [options]

INGESTOR_URL="${INGESTOR_URL:-http://localhost:8081}"
RATE_HZ="${RATE_HZ:-1000}"  # Default 1kHz
SHOT_ID="${SHOT_ID:-}"

usage() {
    cat <<EOF
Usage: $0 <data_file> [options]

Options:
    --ingestor-url URL    Ingestor HTTP endpoint (default: http://localhost:8081)
    --rate-hz RATE        Replay rate in Hz (default: 1000)
    --shot-id ID          Override shot_id from file
    -h, --help           Show this help

The data file should be CSV or JSON with columns:
    - shot_id (or use --shot-id)
    - time_unix_ns or time_ms
    - Channel columns: ip, ne, dwdt, prad, h_alpha, etc.

Example:
    $0 data/shot_12345.csv --rate-hz 500
EOF
    exit 1
}

# Parse arguments
DATA_FILE=""
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
        --shot-id)
            SHOT_ID="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            if [[ -z "$DATA_FILE" ]]; then
                DATA_FILE="$1"
            else
                echo "Unknown option: $1"
                usage
            fi
            shift
            ;;
    esac
done

if [[ -z "$DATA_FILE" ]]; then
    echo "Error: data file required"
    usage
fi

if [[ ! -f "$DATA_FILE" ]]; then
    echo "Error: file not found: $DATA_FILE"
    exit 1
fi

# Check if jq is available for JSON parsing
if command -v jq &> /dev/null; then
    USE_JQ=true
else
    USE_JQ=false
    if [[ "$DATA_FILE" == *.json ]]; then
        echo "Warning: jq not found, JSON parsing may be limited. Install jq for better JSON support."
    fi
fi

# Check if Python is available (for CSV parsing)
if command -v python3 &> /dev/null; then
    USE_PYTHON=true
else
    USE_PYTHON=false
    echo "Warning: python3 not found. CSV parsing may be limited."
fi

# Determine file type
if [[ "$DATA_FILE" == *.json ]]; then
    FORMAT="json"
elif [[ "$DATA_FILE" == *.csv ]]; then
    FORMAT="csv"
else
    echo "Error: unsupported file format. Expected .csv or .json"
    exit 1
fi

echo "Replaying $DATA_FILE to $INGESTOR_URL at ${RATE_HZ}Hz..."

# Calculate delay between samples in nanoseconds
DELAY_NS=$((1000000000 / RATE_HZ))

# Function to send a telemetry point
send_point() {
    local json_data="$1"
    local response
    response=$(curl -s -w "\n%{http_code}" -X POST "$INGESTOR_URL/ingest" \
        -H "Content-Type: application/json" \
        -d "$json_data")
    
    local http_code=$(echo "$response" | tail -n1)
    if [[ "$http_code" != "200" ]]; then
        echo "Error: HTTP $http_code - $(echo "$response" | head -n-1)" >&2
        return 1
    fi
    return 0
}

# Process CSV file
process_csv() {
    if [[ "$USE_PYTHON" == "true" ]]; then
        python3 <<EOF
import csv
import json
import sys
import time
from datetime import datetime

ingestor_url = "$INGESTOR_URL"
delay_ns = $DELAY_NS
shot_id_override = "$SHOT_ID"

with open("$DATA_FILE", "r") as f:
    reader = csv.DictReader(f)
    rows = list(reader)
    
    if not rows:
        print("Error: CSV file is empty", file=sys.stderr)
        sys.exit(1)
    
    # Determine time column
    time_col = None
    for col in ["time_unix_ns", "time_ms", "time"]:
        if col in rows[0]:
            time_col = col
            break
    
    if not time_col:
        print("Error: No time column found (expected time_unix_ns, time_ms, or time)", file=sys.stderr)
        sys.exit(1)
    
    # Get shot_id
    shot_id = shot_id_override if shot_id_override else rows[0].get("shot_id", "unknown")
    
    # Sort by time
    def get_time(row):
        val = float(row[time_col])
        if time_col == "time_ms":
            return int(val * 1_000_000)
        elif time_col == "time":
            return int(val * 1_000_000_000)
        else:
            return int(val)
    
    rows.sort(key=get_time)
    
    # Channel columns (exclude metadata)
    exclude = {"shot_id", "time_unix_ns", "time_ms", "time", "time_to_disruption", "disruption_time_unix_ns"}
    channels = [col for col in rows[0].keys() if col not in exclude]
    
    last_time_ns = None
    sent = 0
    
    for row in rows:
        time_ns = get_time(row)
        
        # Build channels array
        channel_samples = []
        for ch_name in channels:
            try:
                value = float(row[ch_name])
                channel_samples.append({
                    "name": ch_name,
                    "value": value,
                    "quality": "good"
                })
            except (ValueError, KeyError):
                pass
        
        if not channel_samples:
            continue
        
        # Build telemetry point
        point = {
            "shot_id": shot_id,
            "ts_unix_ns": time_ns,
            "channels": channel_samples
        }
        
        # Rate limiting
        if last_time_ns is not None:
            elapsed = time_ns - last_time_ns
            if elapsed < delay_ns:
                sleep_time = (delay_ns - elapsed) / 1_000_000_000
                time.sleep(sleep_time)
        
        # Send point
        json_data = json.dumps(point)
        print(json_data, flush=True)
        
        last_time_ns = time_ns
        sent += 1
        
        if sent % 100 == 0:
            print(f"Sent {sent} points...", file=sys.stderr)
EOF
    else
        echo "Error: Python3 required for CSV processing"
        exit 1
    fi | while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            send_point "$line" || true
        fi
    done
}

# Process JSON file
process_json() {
    if [[ "$USE_JQ" == "true" ]]; then
        # Assume JSON is an array of points or a single point
        jq -c 'if type == "array" then .[] else . end' "$DATA_FILE" | while IFS= read -r point; do
            # Override shot_id if provided
            if [[ -n "$SHOT_ID" ]]; then
                point=$(echo "$point" | jq --arg sid "$SHOT_ID" '.shot_id = $sid')
            fi
            
            send_point "$point"
            
            # Rate limiting - simple sleep
            sleep "$(echo "scale=9; $DELAY_NS / 1000000000" | bc 2>/dev/null || echo "0.001")"
        done
    else
        echo "Error: jq required for JSON processing. Install jq or use Python."
        exit 1
    fi
}

# Main processing
if [[ "$FORMAT" == "csv" ]]; then
    process_csv
else
    process_json
fi

echo ""
echo "Replay completed!"
