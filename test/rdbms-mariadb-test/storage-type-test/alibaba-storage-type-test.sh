#!/bin/bash

# Alibaba Cloud RDBMS StorageType Test Script (MariaDB)
# Fetches StorageTypeOptions from rdbmsmetainfo and runs one test per type in parallel.
# Note: Subnet required.

CSP_NAME="ALIBABA"
CONNECTION_NAME="alibaba-beijing-config"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export RESULT_DIR="${RESULT_DIR:-/tmp/st_results_$$}"
export LOG_DIR="${LOG_DIR:-/tmp/st_logs_$$}"

mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Fetch StorageTypeOptions and DBSpecOptions from metainfo ──────────
echo "[${CSP_NAME}] Fetching StorageTypeOptions from rdbmsmetainfo..."
meta_resp=$(curl -u "${SPIDER_AUTH}" -sX GET \
    "${SPIDER_URL}/spider/rdbmsmetainfo?DBEngine=mariadb&ConnectionName=${CONNECTION_NAME}" 2>&1)

err_msg=$(echo "${meta_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${err_msg}" ]]; then
    echo "[${CSP_NAME}] ERROR fetching metainfo: ${err_msg}"
    echo "${CSP_NAME}|N/A|N/A|FAIL|META_ERROR|-" \
        > "${RESULT_DIR}/result_alibaba_meta_error.txt"
    exit 1
fi

storage_types=$(echo "${meta_resp}" | jq -r '.StorageTypeOptions[]? // empty' 2>/dev/null)
if [[ -z "${storage_types}" ]]; then
    echo "[${CSP_NAME}] No StorageTypeOptions returned - skipping"
    echo "${CSP_NAME}|N/A|N/A|SKIP|NO_STORAGE_TYPES|-" \
        > "${RESULT_DIR}/result_alibaba_skip.txt"
    exit 0
fi

# Pick the first available instance spec from MetaInfo — this ensures the spec is
# actually valid for the region/zone returned by DescribeAvailableZones.
# Using a hardcoded spec risks incompatibility (e.g., rds.mariadb.s4.large is only
# valid for local_ssd; cloud_essd requires a different spec class).
meta_spec=$(echo "${meta_resp}" | jq -r '.DBSpecOptions[0]? // empty' 2>/dev/null)
if [[ -z "${meta_spec}" ]]; then
    meta_spec="rds.mariadb.s4.large"
    echo "[${CSP_NAME}] No DBSpecOptions in metainfo; falling back to ${meta_spec}"
else
    echo "[${CSP_NAME}] Using DBSpec from metainfo: ${meta_spec}"
fi

echo "[${CSP_NAME}] StorageTypeOptions: $(echo "${storage_types}" | tr '\n' ' ')"
echo "[${CSP_NAME}] Launching parallel StorageType tests..."

# ── Launch one test per StorageType ──────────────────────────────────────────
while IFS= read -r storage_type; do
    [[ -z "${storage_type}" ]] && continue

    st_safe=$(echo "${storage_type}" | tr '[:upper:]' '[:lower:]' \
        | sed 's/[^a-z0-9]/-/g' | sed 's/--*/-/g' | sed 's/^-//;s/-$//' | cut -c1-15)
    rdbms_name="cb-mariadb-st-${st_safe}"
    result_file="${RESULT_DIR}/result_alibaba_${st_safe}.txt"
    log_file="${LOG_DIR}/log_alibaba_${st_safe}.txt"

    # Storage size minimums vary by type (Alibaba cloud_essd2 ≥ 500GB, cloud_essd3 ≥ 1500GB).
    case "${storage_type}" in
        cloud_essd2) db_instance_spec="${meta_spec}"; db_storage_size="500"  ;;
        cloud_essd3) db_instance_spec="${meta_spec}"; db_storage_size="1500" ;;
        local_ssd)   db_instance_spec="${meta_spec}"; db_storage_size="20"   ;;
        *)           db_instance_spec="${meta_spec}"; db_storage_size="20"   ;;
    esac

    create_json="{
  \"ConnectionName\": \"${CONNECTION_NAME}\",
  \"ReqInfo\": {
    \"Name\": \"${rdbms_name}\",
    \"VPCName\": \"vpc-01\",
    \"SubnetNames\": [\"subnet-01\"],
    \"DBEngine\": \"mariadb\",
    \"DBEngineVersion\": \"10.6\",
    \"DBSpec\": \"${db_instance_spec}\",
    \"StorageType\": \"${storage_type}\",
    \"StorageSize\": \"${db_storage_size}\",
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

    echo $! > "${LOG_DIR}/pid_alibaba_${st_safe}.txt"
done <<< "${storage_types}"

# ── Wait for all StorageType tests ────────────────────────────────────────────
echo "[${CSP_NAME}] Waiting for all StorageType tests to complete..."
for pid_file in "${LOG_DIR}"/pid_alibaba_*.txt; do
    [[ -f "${pid_file}" ]] || continue
    pid=$(cat "${pid_file}")
    wait "${pid}"
done
echo "[${CSP_NAME}] All StorageType tests completed."
