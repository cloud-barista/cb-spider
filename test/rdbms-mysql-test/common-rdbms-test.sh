#!/bin/bash

# CB-Spider RDBMS Common Test Script
# Flow: Create RDBMS -> Poll until Available -> Get Info -> Write result file
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   RDBMS_NAME      - RDBMS instance name
#   CREATE_JSON     - JSON body for POST /spider/rdbms
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL      - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)
#   MAX_WAIT_SEC    - Max seconds to wait for available status (default: 3600)
#   POLL_INTERVAL   - Polling interval in seconds (default: 30)

format_elapsed() {
    local sec=$1
    if [[ ${sec} -lt 60 ]]; then
        echo "${sec}s"
    else
        echo "$((sec / 60))m$((sec % 60))s"
    fi
}

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
MAX_WAIT_SEC="${MAX_WAIT_SEC:-3600}"
POLL_INTERVAL="${POLL_INTERVAL:-30}"

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

echo "[${CSP_NAME}] [${timestamp}] Creating RDBMS '${RDBMS_NAME}'..."

# ── Create RDBMS ─────────────────────────────────────────────────────────────
create_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/rdbms" \
  -H 'Content-Type: application/json' \
  -d "${CREATE_JSON}" 2>&1)

# Check for API error response
err_msg=$(echo "${create_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${err_msg}" ]]; then
    echo "[${CSP_NAME}] ERROR on create: ${err_msg}"
    echo "${CSP_NAME}|CREATE_ERROR|${err_msg}||||||||-" > "${RESULT_FILE}"
    exit 1
fi

create_status=$(echo "${create_resp}" | jq -r '.Status // .rdbms.Status // "unknown"' 2>/dev/null)
echo "[${CSP_NAME}] Create response status: ${create_status}"

# ── Poll until Available ──────────────────────────────────────────────────────
echo "[${CSP_NAME}] Waiting for RDBMS to become available (poll every ${POLL_INTERVAL}s, max ${MAX_WAIT_SEC}s)..."

elapsed=0
while true; do
    sleep "${POLL_INTERVAL}"
    elapsed=$((elapsed + POLL_INTERVAL))

    poll_resp=$(curl -u "${SPIDER_AUTH}" -s \
      "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}?ConnectionName=${CONNECTION_NAME}" 2>&1)

    cur_status=$(echo "${poll_resp}" | jq -r '.Status // .rdbms.Status // "unknown"' 2>/dev/null)
    status_lower=$(echo "${cur_status}" | tr '[:upper:]' '[:lower:]')

    echo "[${CSP_NAME}] Status: ${cur_status} (elapsed: ${elapsed}s)"

    # Accept various CSP-specific available status strings
    case "${status_lower}" in
        available|ready|runnable|active)
            echo "[${CSP_NAME}] RDBMS is available!"
            break
            ;;
    esac

    if [[ ${elapsed} -ge ${MAX_WAIT_SEC} ]]; then
        echo "[${CSP_NAME}] TIMEOUT: RDBMS did not become available within ${MAX_WAIT_SEC}s"
        echo "${CSP_NAME}|TIMEOUT||||||||||" > "${RESULT_FILE}"
        exit 1
    fi
done

end_time=$(date +%s)
elapsed_total=$((end_time - start_time))

# ── Get RDBMS Info ────────────────────────────────────────────────────────────
info=$(curl -u "${SPIDER_AUTH}" -s \
  "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}?ConnectionName=${CONNECTION_NAME}")

name=$(echo "${info}"       | jq -r '.IId.NameId       // "N/A"')
status=$(echo "${info}"     | jq -r '.Status           // "N/A"')
engine=$(echo "${info}"     | jq -r '.DBEngine         // "N/A"')
version=$(echo "${info}"    | jq -r '.DBEngineVersion  // "N/A"')
spec=$(echo "${info}"       | jq -r '.DBInstanceSpec   // "N/A"')
storage=$(echo "${info}"      | jq -r '.StorageSize  // "N/A"')
storage_type=$(echo "${info}" | jq -r '.StorageType  // "N/A"')
ep_addr=$(echo "${info}"      | jq -r '.Endpoint     // "N/A"')
ep_port=$(echo "${info}"    | jq -r '.Port             // "N/A"')
pub_access=$(echo "${info}" | jq -r '.PublicAccess     // "N/A"')

# Build endpoint display: if Port is valid and not already in Endpoint, append it
if [[ "${ep_port}" != "N/A" && "${ep_port}" != "null" && "${ep_port}" != "" ]]; then
    endpoint_display="${ep_addr}:${ep_port}"
else
    endpoint_display="${ep_addr}"
fi

elapsed_fmt=$(format_elapsed "${elapsed_total}")
echo "[${CSP_NAME}] Done. Endpoint: ${endpoint_display} (total elapsed: ${elapsed_fmt})"

# Ensure result directory exists
mkdir -p "$(dirname "${RESULT_FILE}")"

# Write pipe-separated result line
# Format: CSP|Status|Engine|Version|Spec|Storage(GB)|StorageType|Endpoint|PublicAccess|Elapsed
echo "${CSP_NAME}|${status}|${engine}|${version}|${spec}|${storage}GB|${storage_type}|${endpoint_display}|${pub_access}|${elapsed_fmt}" \
  > "${RESULT_FILE}"
