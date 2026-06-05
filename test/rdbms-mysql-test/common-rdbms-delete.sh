#!/bin/bash

# CB-Spider RDBMS Common Delete Script
# Deletes the RDBMS instance and waits until it is fully removed.
# Author: CB-Spider Team
#
# Required env vars (set by caller):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   RDBMS_NAME      - RDBMS instance name to delete
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL      - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)
#   MAX_WAIT_SEC    - Max seconds to wait for deletion (default: 1800)
#   POLL_INTERVAL   - Polling interval in seconds (default: 15)

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
MAX_WAIT_SEC="${MAX_WAIT_SEC:-1800}"
POLL_INTERVAL="${POLL_INTERVAL:-15}"

format_elapsed() {
    local sec=$1
    if [[ ${sec} -lt 60 ]]; then
        echo "${sec}s"
    else
        echo "$((sec / 60))m$((sec % 60))s"
    fi
}

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

# Ensure result directory exists
mkdir -p "$(dirname "${RESULT_FILE}")"

echo "[${CSP_NAME}] [${timestamp}] Deleting RDBMS '${RDBMS_NAME}'..."

# ── Verify instance exists before deleting ────────────────────────────────────
check_resp=$(curl -u "${SPIDER_AUTH}" -s \
  "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}?ConnectionName=${CONNECTION_NAME}" 2>&1)

err_msg=$(echo "${check_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${err_msg}" ]]; then
    echo "[${CSP_NAME}] Instance not found or error: ${err_msg}"
    echo "${CSP_NAME}|NOT_FOUND|${err_msg}|-" > "${RESULT_FILE}"
    exit 0
fi

# ── Send DELETE request ───────────────────────────────────────────────────────
# Note: DELETE requires ConnectionName in request body, not query parameter
del_resp=$(curl -u "${SPIDER_AUTH}" -sX DELETE \
  "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}" \
  -H 'Content-Type: application/json' \
  -d "{\"ConnectionName\": \"${CONNECTION_NAME}\"}" 2>&1)

del_result=$(echo "${del_resp}" | jq -r '.Result // .result // empty' 2>/dev/null)
del_err=$(echo "${del_resp}"    | jq -r '.message // empty' 2>/dev/null)

if [[ -n "${del_err}" ]]; then
    echo "[${CSP_NAME}] DELETE error: ${del_err}"
    echo "${CSP_NAME}|DELETE_ERROR|${del_err}|-" > "${RESULT_FILE}"
    exit 1
fi

echo "[${CSP_NAME}] DELETE accepted (result: ${del_result:-ok}). Waiting for removal..."

# ── Poll until instance is gone ───────────────────────────────────────────────
elapsed=0
while true; do
    sleep "${POLL_INTERVAL}"
    elapsed=$((elapsed + POLL_INTERVAL))

    poll_resp=$(curl -u "${SPIDER_AUTH}" -s \
      "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}?ConnectionName=${CONNECTION_NAME}" 2>&1)

    # Instance is gone when the API returns an error/message
    poll_err=$(echo "${poll_resp}" | jq -r '.message // empty' 2>/dev/null)
    if [[ -n "${poll_err}" ]]; then
        echo "[${CSP_NAME}] Instance removed (elapsed: ${elapsed}s)"
        break
    fi

    cur_status=$(echo "${poll_resp}" | jq -r '.Status // "unknown"' 2>/dev/null)
    echo "[${CSP_NAME}] Status: ${cur_status} (elapsed: ${elapsed}s)"

    if [[ ${elapsed} -ge ${MAX_WAIT_SEC} ]]; then
        echo "[${CSP_NAME}] TIMEOUT waiting for deletion to complete"
        echo "${CSP_NAME}|DELETE_TIMEOUT||$(format_elapsed ${elapsed})" > "${RESULT_FILE}"
        exit 1
    fi
done

end_time=$(date +%s)
elapsed_total=$((end_time - start_time))
elapsed_fmt=$(format_elapsed "${elapsed_total}")

echo "[${CSP_NAME}] Deletion complete (total elapsed: ${elapsed_fmt})"
echo "${CSP_NAME}|DELETED|ok|${elapsed_fmt}" > "${RESULT_FILE}"
