#!/bin/bash

# Tencent Cloud RDBMS StorageType Test Script
# Fetches StorageTypeOptions from rdbmsmetainfo and runs one test per type in parallel.
# Note: Subnet required. DBInstanceSpec is memory size in MB.
# Note: Tencent may reject concurrent orders (OperationDenied.OtherOderInProcess).
#       The driver handles this with automatic retry logic, so parallel execution is safe.

CSP_NAME="TENCENT"
CONNECTION_NAME="tencent-beijing6-config"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export RESULT_DIR="${RESULT_DIR:-/tmp/st_results_$$}"
export LOG_DIR="${LOG_DIR:-/tmp/st_logs_$$}"

mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Fetch StorageTypeOptions ──────────────────────────────────────────────────
echo "[${CSP_NAME}] Fetching StorageTypeOptions from rdbmsmetainfo..."
meta_resp=$(curl -u "${SPIDER_AUTH}" -sX GET \
    "${SPIDER_URL}/spider/rdbmsmetainfo?DBEngine=mysql&ConnectionName=${CONNECTION_NAME}" 2>&1)

err_msg=$(echo "${meta_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${err_msg}" ]]; then
    echo "[${CSP_NAME}] ERROR fetching metainfo: ${err_msg}"
    echo "${CSP_NAME}|N/A|N/A|FAIL|META_ERROR|-" \
        > "${RESULT_DIR}/result_tencent_meta_error.txt"
    exit 1
fi

storage_types=$(echo "${meta_resp}" | jq -r '.StorageTypeOptions[]? // empty' 2>/dev/null)
if [[ -z "${storage_types}" ]]; then
    echo "[${CSP_NAME}] No StorageTypeOptions returned - skipping"
    echo "${CSP_NAME}|N/A|N/A|SKIP|NO_STORAGE_TYPES|-" \
        > "${RESULT_DIR}/result_tencent_skip.txt"
    exit 0
fi

echo "[${CSP_NAME}] StorageTypeOptions: $(echo "${storage_types}" | tr '\n' ' ')"
echo "[${CSP_NAME}] Launching parallel StorageType tests..."

# ── Launch one test per StorageType in parallel ───────────────────────────────
while IFS= read -r storage_type; do
    [[ -z "${storage_type}" ]] && continue

    st_safe=$(echo "${storage_type}" | tr '[:upper:]' '[:lower:]' \
        | sed 's/[^a-z0-9]/-/g' | sed 's/--*/-/g' | sed 's/^-//;s/-$//' | cut -c1-15)
    rdbms_name="cb-mysql-st-${st_safe}"
    result_file="${RESULT_DIR}/result_tencent_${st_safe}.txt"
    log_file="${LOG_DIR}/log_tencent_${st_safe}.txt"

    create_json="{
  \"ConnectionName\": \"${CONNECTION_NAME}\",
  \"ReqInfo\": {
    \"Name\": \"${rdbms_name}\",
    \"VPCName\": \"vpc-01\",
    \"SubnetNames\": [\"subnet-01\"],
    \"DBEngine\": \"mysql\",
    \"DBEngineVersion\": \"8.0\",
    \"DBInstanceSpec\": \"8000\",
    \"StorageType\": \"${storage_type}\",
    \"StorageSize\": \"50\",
    \"MasterUserName\": \"root\",
    \"MasterUserPassword\": \"Password123!\",
    \"PublicAccess\": true
  }
}"

    echo "[${CSP_NAME}] Launching test: StorageType='${storage_type}' (RDBMS: ${rdbms_name})"
    (
        export CSP_NAME="${CSP_NAME}"
        export CONNECTION_NAME="${CONNECTION_NAME}"
        export RDBMS_NAME="${rdbms_name}"
        export STORAGE_TYPE="${storage_type}"
        export CREATE_JSON="${create_json}"
        export RESULT_FILE="${result_file}"
        exec "${SCRIPT_DIR}/common-storage-type-test.sh"
    ) > "${log_file}" 2>&1 &

    echo $! > "${LOG_DIR}/pid_tencent_${st_safe}.txt"
done <<< "${storage_types}"

# ── Wait for all StorageType tests ────────────────────────────────────────────
echo "[${CSP_NAME}] Waiting for all StorageType tests to complete..."
for pid_file in "${LOG_DIR}"/pid_tencent_*.txt; do
    [[ -f "${pid_file}" ]] || continue
    pid=$(cat "${pid_file}")
    wait "${pid}"
done
echo "[${CSP_NAME}] All StorageType tests completed."
