#!/bin/bash

# Set default RESTSERVER if not set
if [[ -z "$RESTSERVER" ]]; then
  echo "RESTSERVER is not set. Defaulting to 'localhost'."
  export RESTSERVER="localhost"
fi

echo "####################################################################"
echo "## Starting cloud unregistration process for all providers"
echo "####################################################################"

# Find all delete scripts and run them in parallel
DELETE_SCRIPTS=$(find . -name "*_delete.sh" -not -name "all_delete.sh" | sort)

# Create a temp file to capture completion status
TMP_DONE_LOG=$(mktemp)

# Store all background PIDs
PIDS=()

for script in $DELETE_SCRIPTS; do
  provider=$(basename "$script" | sed 's/_delete.sh//')
  echo "[START] Unregistering $provider"
  
  # Run in background and capture PID
  bash "$script" > /dev/null 2>&1 &
  PID=$!
  PIDS+=($PID)
  
  # Store mapping of PID to provider name for status reporting
  echo "$PID:$provider" >> "$TMP_DONE_LOG.map"
done

echo "Started $(echo ${PIDS[@]} | wc -w | tr -d ' ') unregistration processes"
echo "Waiting for completion (this might take a few minutes)..."

# Wait for each PID with a timeout
for PID in "${PIDS[@]}"; do
  PROVIDER=$(grep "^$PID:" "$TMP_DONE_LOG.map" | cut -d':' -f2)
  
  # Set a timeout of 240 seconds for each process
  TIMEOUT=240
  COUNT=0
  
  while kill -0 $PID 2>/dev/null; do
    sleep 1
    COUNT=$((COUNT+1))
    
    # Check if we've exceeded the timeout
    if [ $COUNT -ge $TIMEOUT ]; then
      echo "[TIMEOUT] $PROVIDER unregistration timed out after ${TIMEOUT}s - killing process"
      kill -9 $PID 2>/dev/null
      echo "$PROVIDER:TIMEOUT" >> "$TMP_DONE_LOG"
      break
    fi
  done
  
  # If process completed within timeout
  if [ $COUNT -lt $TIMEOUT ]; then
    # Check if process exited successfully
    wait $PID
    STATUS=$?
    if [ $STATUS -eq 0 ]; then
      echo "$PROVIDER:SUCCESS" >> "$TMP_DONE_LOG"
    else
      echo "$PROVIDER:FAILED (exit code $STATUS)" >> "$TMP_DONE_LOG"
    fi
  fi
done

echo "####################################################################"
echo "## Unregistration Status"
echo "####################################################################"
cat "$TMP_DONE_LOG" | sort | while IFS=':' read -r PROVIDER STATUS; do
  echo "[$STATUS] $PROVIDER"
done

# Clean up temp files
rm -f "$TMP_DONE_LOG" "$TMP_DONE_LOG.map"

echo "####################################################################"
echo "## All cloud provider unregistration processes completed"
echo "####################################################################"

