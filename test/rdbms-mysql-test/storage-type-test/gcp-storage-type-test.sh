#!/bin/bash

# GCP RDBMS StorageType Test Script
# Tests 5 fixed combinations of Storage Option, Edition, and Machine Series:
#
#   Storage Option     | Edition        | Machine Series
#   -------------------|----------------|-----------------------------
#   PD_SSD             | Enterprise Plus| N2  (db-perf-optimized-N-4)
#   PD_SSD             | Enterprise     | Dedicated core (db-custom-2-8192)
#   PD_HDD             | Enterprise     | Dedicated core (db-custom-2-8192)
#   HYPERDISK_BALANCED | Enterprise Plus| C4A (db-c4a-highmem-4)
#   HYPERDISK_BALANCED | Enterprise     | N4  (db-custom-N4-2-4096)

CSP_NAME="GCP"
CONNECTION_NAME="gcp-iowa-config"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export RESULT_DIR="${RESULT_DIR:-/tmp/st_results_$$}"
export LOG_DIR="${LOG_DIR:-/tmp/st_logs_$$}"

mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Test cases: storage_type|db_instance_spec|storage_size_gb|rdbms_suffix ──
# rdbms_suffix is used as part of the instance name: cb-mysql-st-<suffix>
TEST_CASES=(
    "PD_SSD|db-perf-optimized-N-4|10|pdssd-n2"
    "PD_SSD|db-custom-2-8192|10|pdssd-ent"
    "PD_HDD|db-custom-2-8192|10|pdhdd-ent"
    "HYPERDISK_BALANCED|db-c4a-highmem-4|20|hdb-c4a"
    "HYPERDISK_BALANCED|db-custom-N4-2-4096|20|hdb-n4"
)

echo "[${CSP_NAME}] Launching ${#TEST_CASES[@]} StorageType tests in parallel..."

for test_case in "${TEST_CASES[@]}"; do
    IFS='|' read -r storage_type db_instance_spec db_storage_size rdbms_suffix <<< "${test_case}"

    rdbms_name="cb-mysql-st-${rdbms_suffix}"
    result_file="${RESULT_DIR}/result_gcp_${rdbms_suffix}.txt"
    log_file="${LOG_DIR}/log_gcp_${rdbms_suffix}.txt"

    create_json="{
  \"ConnectionName\": \"${CONNECTION_NAME}\",
  \"ReqInfo\": {
    \"Name\": \"${rdbms_name}\",
    \"VPCName\": \"vpc-01\",
    \"DBEngine\": \"mysql\",
    \"DBEngineVersion\": \"8.0\",
    \"DBInstanceSpec\": \"${db_instance_spec}\",
    \"StorageType\": \"${storage_type}\",
    \"StorageSize\": \"${db_storage_size}\",
    \"MasterUserName\": \"myadmin\",
    \"MasterUserPassword\": \"Password123!\",
    \"PublicAccess\": true
  }
}"

    echo "[${CSP_NAME}] Launching: StorageType='${storage_type}' Spec='${db_instance_spec}' Size=${db_storage_size}GB (RDBMS: ${rdbms_name})"
    (
        export CSP_NAME="${CSP_NAME}"
        export CONNECTION_NAME="${CONNECTION_NAME}"
        export RDBMS_NAME="${rdbms_name}"
        export STORAGE_TYPE="${storage_type}"
        export CREATE_JSON="${create_json}"
        export RESULT_FILE="${result_file}"
        exec "${SCRIPT_DIR}/common-storage-type-test.sh"
    ) > "${log_file}" 2>&1 &

    echo $! > "${LOG_DIR}/pid_gcp_${rdbms_suffix}.txt"
done

# ── Wait for all tests ────────────────────────────────────────────────────────
echo "[${CSP_NAME}] Waiting for all StorageType tests to complete..."
for pid_file in "${LOG_DIR}"/pid_gcp_*.txt; do
    [[ -f "${pid_file}" ]] || continue
    pid=$(cat "${pid_file}")
    wait "${pid}"
done
echo "[${CSP_NAME}] All StorageType tests completed."
