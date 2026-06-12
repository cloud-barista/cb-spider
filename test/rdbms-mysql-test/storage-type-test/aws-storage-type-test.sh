#!/bin/bash

# AWS RDBMS StorageType Test Script
# Fetches StorageTypeOptions from rdbmsmetainfo and runs one test per type in parallel.
# Note: io1/io2 require StorageSize >= 100. SubnetGroup requires 2+ subnets in different AZs.

CSP_NAME="AWS"
CONNECTION_NAME="aws-config01"
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
        > "${RESULT_DIR}/result_aws_meta_error.txt"
    exit 1
fi

storage_types=$(echo "${meta_resp}" | jq -r '.StorageTypeOptions[]? // empty' 2>/dev/null)
if [[ -z "${storage_types}" ]]; then
    echo "[${CSP_NAME}] No StorageTypeOptions returned - skipping"
    echo "${CSP_NAME}|N/A|N/A|SKIP|NO_STORAGE_TYPES|-" \
        > "${RESULT_DIR}/result_aws_skip.txt"
    exit 0
fi

echo "[${CSP_NAME}] StorageTypeOptions: $(echo "${storage_types}" | tr '\n' ' ')"
echo "[${CSP_NAME}] Launching parallel StorageType tests..."

# ── Launch one test per StorageType ──────────────────────────────────────────
while IFS= read -r storage_type; do
    [[ -z "${storage_type}" ]] && continue

    st_safe=$(echo "${storage_type}" | tr '[:upper:]' '[:lower:]' \
        | sed 's/[^a-z0-9]/-/g' | sed 's/--*/-/g' | sed 's/^-//;s/-$//' | cut -c1-15)
    rdbms_name="cb-mysql-st-${st_safe}"
    result_file="${RESULT_DIR}/result_aws_${st_safe}.txt"
    log_file="${LOG_DIR}/log_aws_${st_safe}.txt"

    # AWS per-StorageType configuration:
    #   io1/io2: Iops required (100-64000); StorageSize >= 100
    #   gp3:     Iops not specified (AWS applies default 3000 IOPS automatically)
    #   others:  no extra params
    case "${storage_type}" in
        io1|io2) iops_field="\"Iops\": \"3000\"," ; storage_size="100" ;;
        *)       iops_field=""                    ; storage_size="100" ;;
    esac

    create_json="{
  \"ConnectionName\": \"${CONNECTION_NAME}\",
  \"ReqInfo\": {
    \"Name\": \"${rdbms_name}\",
    \"VPCName\": \"vpc-01\",
    \"DBEngine\": \"mysql\",
    \"DBEngineVersion\": \"8.0\",
    \"DBInstanceSpec\": \"db.t3.medium\",
    \"StorageType\": \"${storage_type}\",
    \"StorageSize\": \"${storage_size}\",
    ${iops_field}
    \"SubnetNames\": [\"subnet-01\", \"subnet-02\"],
    \"SecurityGroupNames\": [\"sg-01\"],
    \"MasterUserName\": \"myadmin\",
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

    echo $! > "${LOG_DIR}/pid_aws_${st_safe}.txt"
done <<< "${storage_types}"

# ── Wait for all StorageType tests ────────────────────────────────────────────
echo "[${CSP_NAME}] Waiting for all StorageType tests to complete..."
for pid_file in "${LOG_DIR}"/pid_aws_*.txt; do
    [[ -f "${pid_file}" ]] || continue
    pid=$(cat "${pid_file}")
    wait "${pid}"
done
echo "[${CSP_NAME}] All StorageType tests completed."
