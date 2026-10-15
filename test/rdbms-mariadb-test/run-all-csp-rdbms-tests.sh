#!/bin/bash

# CB-Spider RDBMS Test Runner for All CSPs (MariaDB)
# Runs RDBMS creation on the CSPs that support MariaDB (AWS, Alibaba,
# OpenStack, NHN) in parallel, waits for each to become available, then
# collects and displays a unified result table.
# Azure/GCP/IBM/NCP/Tencent do not support MariaDB and are excluded.
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export MAX_WAIT_SEC="${MAX_WAIT_SEC:-3600}"   # 60 min timeout per CSP
export POLL_INTERVAL="${POLL_INTERVAL:-30}"   # poll every 30s

# Temp directories
export RESULT_DIR="/tmp/rdbms_results_$$"
LOG_DIR="/tmp/rdbms_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-rdbms-test.sh"       ;;
        ALIBABA)   echo "alibaba-rdbms-test.sh"   ;;
        OPENSTACK) echo "openstack-rdbms-test.sh" ;;
        NHN)       echo "nhn-rdbms-test.sh"       ;;
    esac
}

print_separator() {
    printf '%177s\n' '' | tr ' ' '-'
}

print_header() {
    echo ""
    printf '%177s\n' '' | tr ' ' '='
    echo "                                           RDBMS CREATE & INFO TEST SUMMARY - ALL CSPs (MariaDB)"
    printf '%177s\n' '' | tr ' ' '='
    echo ""
    printf "%-12s | %-11s | %-8s | %-12s | %-24s | %-24s | %-40s | %-12s | %-10s\n" \
        "CSP" "Status" "Engine" "Version" "Spec" "Storage" "Endpoint" "PublicAccess" "Elapsed"
    print_separator
}

# ── Run all CSP tests in parallel ────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#       CB-Spider RDBMS Multi-CSP Test (MariaDB) - Starting All CSPs          #"
echo "################################################################################"
echo ""
echo "Spider URL   : ${SPIDER_URL}"
echo "Max wait     : ${MAX_WAIT_SEC}s per CSP"
echo "Poll interval: ${POLL_INTERVAL}s"
echo "Result dir   : ${RESULT_DIR}"
echo "Log dir      : ${LOG_DIR}"
echo ""
echo "Launching RDBMS creation on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS ALIBABA OPENSTACK NHN"

# Launch all CSP scripts in background; store PIDs in /tmp files (bash 3.2 compatible)
for csp in ${CSP_ORDER}; do
    script=$(csp_script "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} test (log: ${log_file})"
    "${SCRIPT_DIR}/${script}" > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_${csp}.txt"
done

echo ""
echo "[MAIN] All CSP tests launched. Waiting for completion..."
echo "[MAIN] Monitor progress: tail -f ${LOG_DIR}/log_<csp_lowercase>.txt"
echo ""

# ── Wait for all background jobs ─────────────────────────────────────────────
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

fail_count=0

for csp in ${CSP_ORDER}; do
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    if [[ -f "${result_file}" ]]; then
        IFS='|' read -r r_csp r_status r_engine r_version r_spec r_storage r_storage_type r_endpoint r_public r_elapsed \
            < "${result_file}"
    else
        r_csp="${csp}"
        r_status="NO_RESULT"
        r_engine="-"
        r_version="-"
        r_spec="-"
        r_storage="-"
        r_storage_type="-"
        r_endpoint="-"
        r_public="-"
        r_elapsed="-"
    fi

    r_storage_display="${r_storage}|${r_storage_type}"
    printf "%-12s | %-11s | %-8s | %-12s | %-24s | %-24s | %-40s | %-12s | %-10s\n" \
        "${r_csp}" "${r_status}" "${r_engine}" "${r_version}" \
        "${r_spec}" "${r_storage_display}" "${r_endpoint}" "${r_public}" "${r_elapsed}"

    r_status_lower=$(echo "${r_status}" | tr '[:upper:]' '[:lower:]')
    case "${r_status_lower}" in
        available|ready|runnable|active) ;;
        *) fail_count=$((fail_count + 1)) ;;
    esac
done

print_separator
echo ""
echo "Failed : ${fail_count}"
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
printf '%177s\n' '' | tr ' ' '='
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

[[ ${fail_count} -eq 0 ]]
