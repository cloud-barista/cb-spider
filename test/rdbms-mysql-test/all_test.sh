#!/bin/bash

# CB-Spider RDBMS Full Test Suite
# Runs all RDBMS tests in sequence:
#   1. Network Prepare   - VPC/Subnet/SG 사전 생성
#   2. StorageType Test  - CSP별 StorageType 생성 검증
#   3. StorageType RDBMS Delete
#   4. RDBMS Create      - 9개 CSP 인스턴스 생성
#   5. Database Test     - 인스턴스 내부 DB CRUD 검증
#   6. Tag Test          - SupportsTag=true CSP 태그 CRUD 검증
#   7. RDBMS Delete      - 9개 CSP 인스턴스 삭제
#   8. Network Cleanup   - VPC/Subnet/SG 삭제
# Author: CB-Spider Team

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

PASS_COUNT=0
FAIL_COUNT=0
declare -a STEP_RESULTS

run_step() {
    local step_num="$1"
    local step_name="$2"
    local script="$3"

    echo ""
    echo "############################################################"
    printf "#  Step %-2s: %-48s  #\n" "${step_num}" "${step_name}"
    echo "############################################################"
    echo ""

    bash "${script}"
    local exit_code=$?

    if [[ ${exit_code} -eq 0 ]]; then
        STEP_RESULTS+=("PASS | Step ${step_num}: ${step_name}")
        PASS_COUNT=$((PASS_COUNT + 1))
        echo ""
        echo "[OK] Step ${step_num} completed successfully."
    else
        STEP_RESULTS+=("FAIL | Step ${step_num}: ${step_name} (exit code: ${exit_code})")
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo ""
        echo "[FAIL] Step ${step_num} finished with exit code ${exit_code}."
        if [[ "${STOP_ON_FAIL:-0}" == "1" ]]; then
            echo "[ABORT] STOP_ON_FAIL=1 — stopping test suite."
            print_summary
            exit 1
        fi
    fi
}

print_summary() {
    echo ""
    echo "############################################################"
    echo "#                   FULL TEST SUITE SUMMARY                #"
    echo "############################################################"
    echo ""
    for result in "${STEP_RESULTS[@]}"; do
        echo "  ${result}"
    done
    echo ""
    printf "  Total: %d PASS, %d FAIL\n" "${PASS_COUNT}" "${FAIL_COUNT}"
    echo ""
    echo "############################################################"
    echo ""
}

echo ""
echo "############################################################"
echo "#         CB-Spider RDBMS Full Test Suite - START          #"
echo "############################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "STOP_ON_FAIL: ${STOP_ON_FAIL:-0} (set to 1 to abort on first failure)"
echo ""

# ── Step 1: Network Prepare ───────────────────────────────────────────────────
run_step 1 "Network Prepare (VPC/Subnet/SG)" \
    "${SCRIPT_DIR}/run-all-csp-network-prepare.sh"

# ── Step 2: StorageType Test ──────────────────────────────────────────────────
run_step 2 "StorageType Validation Test" \
    "${SCRIPT_DIR}/storage-type-test/run-all-csp-storage-type-tests.sh"

# ── Step 3: Delete StorageType RDBMS instances ───────────────────────────────
run_step 3 "Delete StorageType RDBMS Instances" \
    "${SCRIPT_DIR}/storage-type-test/delete-all-csp-storage-type-rdbms.sh"

# ── Step 4: RDBMS Create ──────────────────────────────────────────────────────
run_step 4 "RDBMS Instance Create (all CSPs)" \
    "${SCRIPT_DIR}/run-all-csp-rdbms-tests.sh"

# ── Step 5: Database Management Test ─────────────────────────────────────────
run_step 5 "Database Management Test (CRUD inside instance)" \
    "${SCRIPT_DIR}/database-test/run-all-csp-database-tests.sh"

# ── Step 6: Tag Management Test ───────────────────────────────────────────────
run_step 6 "Tag Management Test (SupportsTag=true CSPs)" \
    "${SCRIPT_DIR}/tag-test/run-all-csp-rdbms-tag-tests.sh"

# ── Step 7: RDBMS Delete ──────────────────────────────────────────────────────
run_step 7 "RDBMS Instance Delete (all CSPs)" \
    "${SCRIPT_DIR}/delete-all-csp-rdbms.sh"

# ── Step 8: Network Cleanup ───────────────────────────────────────────────────
run_step 8 "Network Cleanup (VPC/Subnet/SG)" \
    "${SCRIPT_DIR}/delete-all-csp-network.sh"

# ── Final Summary ─────────────────────────────────────────────────────────────
print_summary

[[ ${FAIL_COUNT} -eq 0 ]]
