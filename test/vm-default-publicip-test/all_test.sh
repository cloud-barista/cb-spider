#!/bin/bash

# CB-Spider VM Default-PublicIP Full Test Suite
# Runs the full flow in sequence:
#   1. Basic Resource Prepare - VPC/Subnet -> SecurityGroup -> KeyPair
#   2. VM PublicIP Test       - Create -> wait Running -> record PublicIP
#                               -> Suspend -> record PublicIP
#                               -> Resume  -> record PublicIP -> compare
#   3. VM Delete              - Terminate the 10 CSP VM instances
#   4. Basic Resource Cleanup - KeyPair -> SecurityGroup -> VPC/Subnet
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
echo "#     CB-Spider VM Default-PublicIP Full Test Suite - START #"
echo "############################################################"
echo ""
echo "Spider URL  : ${SPIDER_URL}"
echo "STOP_ON_FAIL: ${STOP_ON_FAIL:-0} (set to 1 to abort on first failure)"
echo ""

# ── Step 1: Basic Resource Prepare ───────────────────────────────────────────
run_step 1 "Basic Resource Prepare (VPC/Subnet/SG/KeyPair)" \
    "${SCRIPT_DIR}/run-all-csp-network-prepare.sh"

# ── Step 2: VM PublicIP Test ──────────────────────────────────────────────────
run_step 2 "VM PublicIP Test: Create/Suspend/Resume (all CSPs)" \
    "${SCRIPT_DIR}/run-all-csp-vm-publicip-tests.sh"

# ── Step 3: VM Delete ─────────────────────────────────────────────────────────
run_step 3 "VM Delete (all CSPs)" \
    "${SCRIPT_DIR}/delete-all-csp-vm.sh"

# ── Step 4: Basic Resource Cleanup ────────────────────────────────────────────
run_step 4 "Basic Resource Cleanup (KeyPair/SG/VPC)" \
    "${SCRIPT_DIR}/delete-all-csp-network.sh"

# ── Final Summary ─────────────────────────────────────────────────────────────
print_summary

[[ ${FAIL_COUNT} -eq 0 ]]
