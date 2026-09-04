#!/bin/bash

# CB-Spider VM AssignPublicIP=true Test Runner for All CSPs
# For each CSP: creates a VM with ReqInfo.AssignPublicIP=true, waits for
# Running, verifies PrivateIP AND PublicIP are both set, SSH-checks the
# initial PublicIP, then exercises the default-PublicIP lifecycle in reverse:
# UnassignVMDefaultPublicIP -> verify PublicIP is now empty ->
# AssignVMDefaultPublicIP -> verify PublicIP is set again -> SSH check via
# the re-assigned PublicIP.
# FAIL on any step failing (VM never reaches Running, a PublicIP present/absent
# when it shouldn't be, Assign/Unassign API errors, or an SSH login failure).
# All CSPs run concurrently. A unified result table is shown at the end.
#
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export MAX_WAIT_SEC="${MAX_WAIT_SEC:-1800}"
export POLL_INTERVAL="${POLL_INTERVAL:-15}"

BASE_DIR="/tmp/vm_truepublicip_test_$$"
export RESULT_DIR="${BASE_DIR}/results"
LOG_DIR="${BASE_DIR}/logs"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-assign-publicip-true-test.sh"       ;;
        AZURE)     echo "azure-assign-publicip-true-test.sh"     ;;
        GCP)       echo "gcp-assign-publicip-true-test.sh"       ;;
        ALIBABA)   echo "alibaba-assign-publicip-true-test.sh"   ;;
        TENCENT)   echo "tencent-assign-publicip-true-test.sh"   ;;
        IBM)       echo "ibm-assign-publicip-true-test.sh"       ;;
        OPENSTACK) echo "openstack-assign-publicip-true-test.sh" ;;
        NCP)       echo "ncp-assign-publicip-true-test.sh"       ;;
        NHN)       echo "nhn-assign-publicip-true-test.sh"       ;;
        KT)        echo "kt-assign-publicip-true-test.sh"        ;;
    esac
}

print_separator() {
    printf '%194s\n' '' | tr ' ' '-'
}

print_header() {
    echo ""
    printf '%194s\n' '' | tr ' ' '='
    echo "                    VM AssignPublicIP=true TEST SUMMARY - ALL CSPs"
    printf '%194s\n' '' | tr ' ' '='
    echo ""
    printf "%-11s | %-6s | %-10s | %-15s | %-15s | %-6s | %-15s | %-15s | %-6s | %-8s | %-s\n" \
        "CSP" "Result" "VMStatus" "PrivateIP" "PubIP(init)" "SSH" "PubIP(unassign)" "PubIP(reassign)" "SSH2" "Elapsed" "Reason"
    print_separator
}

# ── Banner ─────────────────────────────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#     CB-Spider VM AssignPublicIP=true Test - Starting All CSPs               #"
echo "################################################################################"
echo ""
echo "Spider URL   : ${SPIDER_URL}"
echo "Max wait     : ${MAX_WAIT_SEC}s per CSP"
echo "Poll interval: ${POLL_INTERVAL}s"
echo "Base dir     : ${BASE_DIR}"
echo ""
echo "Launching all CSP tests in parallel..."
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN KT"

# ── Launch all CSP scripts in background ─────────────────────────────────────
for csp in ${CSP_ORDER}; do
    script=$(csp_script "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} (log: ${log_file})"
    "${SCRIPT_DIR}/${script}" > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_${csp}.txt"
done

echo ""
echo "[MAIN] All CSP tests launched. Waiting for completion..."
echo "[MAIN] Monitor: tail -f ${LOG_DIR}/log_<csp>.txt"
echo ""

# ── Wait for all CSP background jobs ─────────────────────────────────────────
for csp in ${CSP_ORDER}; do
    pid=$(cat "${LOG_DIR}/pid_${csp}.txt" 2>/dev/null)
    if [[ -n "${pid}" ]]; then
        wait "${pid}"
        exit_code=$?
        if [[ ${exit_code} -eq 0 ]]; then
            echo "[MAIN] ${csp} completed successfully"
        else
            echo "[MAIN] ${csp} finished with exit code ${exit_code} (check ${LOG_DIR}/log_$(to_lower "${csp}").txt)"
        fi
    fi
done

echo ""
echo "[MAIN] All CSP tests finished. Collecting results..."
echo ""

# ── Print result table ────────────────────────────────────────────────────────
print_header

pass_count=0
fail_count=0

for csp in ${CSP_ORDER}; do
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    if [[ -f "${result_file}" ]]; then
        IFS='|' read -r r_csp r_result r_status r_priv r_pub_init r_ssh_init r_pub_unassign r_pub_reassign r_ssh_reassign r_elapsed r_reason \
            < "${result_file}"
    else
        r_csp="${csp}"
        r_result="NO_RESULT"
        r_status="-"
        r_priv="-"
        r_pub_init="-"
        r_ssh_init="-"
        r_pub_unassign="-"
        r_pub_reassign="-"
        r_ssh_reassign="-"
        r_elapsed="-"
        r_reason="script crashed before writing a result"
    fi

    printf "%-11s | %-6s | %-10s | %-15s | %-15s | %-6s | %-15s | %-15s | %-6s | %-8s | %-s\n" \
        "${r_csp}" "${r_result}" "${r_status}" "${r_priv}" "${r_pub_init}" "${r_ssh_init}" "${r_pub_unassign}" "${r_pub_reassign}" "${r_ssh_reassign}" "${r_elapsed}" "${r_reason}"

    if [[ "${r_result}" == "PASS" ]]; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
done

print_separator
echo ""
echo "Total: $((pass_count + fail_count))  PASS: ${pass_count}  FAIL: ${fail_count}"
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
echo "Legend : PASS requires, in order: VMStatus=Running with a non-empty"
echo "         PrivateIP and a non-empty PublicIP right after create (since"
echo "         AssignPublicIP=true), SSH-reachable via that initial PublicIP;"
echo "         UnassignVMDefaultPublicIP succeeds and PublicIP goes back to"
echo "         empty; AssignVMDefaultPublicIP results in a non-empty PublicIP"
echo "         again that is SSH-reachable. Any step failing (create error,"
echo "         timeout, VM entered Failed, PublicIP present/absent when it"
echo "         shouldn't be, Assign/Unassign API errors, or SSH not confirmed"
echo "         OK) is FAIL."
echo "         SSH columns: OK (login succeeded) is the only passing value - FAIL"
echo "         (login failed after retries), NO_IP, and NO_KEY (PrivateKey file"
echo "         missing, see KEY_DIR) all fail the test since reachability could"
echo "         not be confirmed."
echo ""
printf '%194s\n' '' | tr ' ' '='
echo ""

# ── Per-CSP full log dump (optional, controlled by VERBOSE=1) ────────────────
if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo ""
    echo "################################################################################"
    echo "#                          Per-CSP Detailed Logs                              #"
    echo "################################################################################"
    for csp in ${CSP_ORDER}; do
        log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
        echo ""
        echo "────────────────────────────── ${csp} ──────────────────────────────"
        if [[ -f "${log_file}" ]]; then
            cat "${log_file}"
        else
            echo "(no log)"
        fi
    done
fi

# Propagate failure to caller so a non-zero FAIL count fails this step
[[ ${fail_count} -eq 0 ]]
