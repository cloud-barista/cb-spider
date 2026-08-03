#!/bin/bash

# CB-Spider RDBMS Database Management Test Runner for All CSPs
# Tests CreateDatabase / ListDatabases / DeleteDatabase inside RDBMS instances.
# Prerequisite: RDBMS instances must already be created (run ../run-all-csp-rdbms-tests.sh first).
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

export RESULT_DIR="/tmp/rdbms_mgmt_results_$$"
LOG_DIR="/tmp/rdbms_mgmt_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-database-test.sh"       ;;
        AZURE)     echo "azure-database-test.sh"     ;;
        GCP)       echo "gcp-database-test.sh"       ;;
        ALIBABA)   echo "alibaba-database-test.sh"   ;;
        TENCENT)   echo "tencent-database-test.sh"   ;;
        IBM)       echo "ibm-database-test.sh"       ;;
        OPENSTACK) echo "openstack-database-test.sh" ;;
        NCP)       echo "ncp-database-test.sh"       ;;
        NHN)       echo "nhn-database-test.sh"       ;;
    esac
}

print_separator() {
    printf '%103s\n' '' | tr ' ' '-'
}

print_header() {
    echo ""
    printf '%103s\n' '' | tr ' ' '='
    echo "                  RDBMS DATABASE MANAGEMENT TEST SUMMARY - ALL CSPs"
    printf '%103s\n' '' | tr ' ' '='
    echo ""
    printf "%-12s | %-10s | %-8s | %-12s | %-10s | %-13s | %-10s\n" \
        "CSP" "CreateDB" "ListDB" "FoundInList" "DeleteDB" "VerifyDeleted" "Elapsed"
    print_separator
}

# ── Launch ────────────────────────────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#    CB-Spider RDBMS Database Management Multi-CSP Test - Starting All CSPs   #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Launching database management tests on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN"

for csp in ${CSP_ORDER}; do
    script=$(csp_script "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} (log: ${log_file})"
    "${SCRIPT_DIR}/${script}" > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_${csp}.txt"
done

echo ""
echo "[MAIN] All tests launched. Waiting for completion..."
echo "[MAIN] Monitor progress: tail -f ${LOG_DIR}/log_<csp_lowercase>.txt"
echo ""

# ── Wait for all ──────────────────────────────────────────────────────────────
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
echo "[MAIN] All tests finished. Collecting results..."
echo ""

# ── Print result table ────────────────────────────────────────────────────────
print_header

pass_count=0
fail_count=0

for csp in ${CSP_ORDER}; do
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    if [[ -f "${result_file}" ]]; then
        IFS='|' read -r r_csp r_create r_list r_found r_delete r_verify r_elapsed \
            < "${result_file}"
    else
        r_csp="${csp}"
        r_create="NO_RESULT"
        r_list="-"
        r_found="-"
        r_delete="-"
        r_verify="-"
        r_elapsed="-"
    fi

    printf "%-12s | %-10s | %-8s | %-12s | %-10s | %-13s | %-10s\n" \
        "${r_csp}" "${r_create}" "${r_list}" "${r_found}" "${r_delete}" "${r_verify}" "${r_elapsed}"

    if [[ "${r_create}" == "PASS" && "${r_list}" == "PASS" && "${r_found}" == "FOUND" && \
          "${r_delete}" == "PASS" && "${r_verify}" == "PASS" ]]; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
done

print_separator
echo ""
printf "Total: %d PASS, %d FAIL\n" "${pass_count}" "${fail_count}"
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
printf '%103s\n' '' | tr ' ' '='
echo ""

if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo ""
    echo "################################################################################"
    echo "#                          Per-CSP Detailed Logs                              #"
    echo "################################################################################"
    for csp in ${CSP_ORDER}; do
        log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
        echo ""
        echo "────────────────────────────── ${csp} ──────────────────────────────"
        [[ -f "${log_file}" ]] && cat "${log_file}" || echo "(no log)"
    done
fi

# Propagate failure to caller (all_test.sh) so a nonzero FAIL count fails this step
[[ ${fail_count} -eq 0 ]]
