#!/bin/bash

# CB-Spider RDBMS StorageType Test Runner - All CSPs (MariaDB)
# For each CSP:
#   1. Fetches StorageTypeOptions via rdbmsmetainfo API (DBEngine=mariadb)
#   2. Creates one RDBMS per StorageType option (in parallel within each CSP)
#   3. Verifies returned StorageType matches requested
#   4. Deletes test instances after verification
# Only CSPs that support MariaDB (AWS, Alibaba, OpenStack, NHN) are run;
# Azure/GCP/IBM/NCP/Tencent do not support MariaDB and are excluded entirely.
# All CSPs run concurrently. A unified result table is shown at the end.
#
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export MAX_WAIT_SEC="${MAX_WAIT_SEC:-3600}"
export POLL_INTERVAL="${POLL_INTERVAL:-30}"
export AUTO_DELETE="${AUTO_DELETE:-false}"

BASE_DIR="/tmp/st_test_$$"
export RESULT_DIR="${BASE_DIR}/results"
export LOG_DIR="${BASE_DIR}/logs"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-storage-type-test.sh"       ;;
        ALIBABA)   echo "alibaba-storage-type-test.sh"   ;;
        OPENSTACK) echo "openstack-storage-type-test.sh" ;;
        NHN)       echo "nhn-storage-type-test.sh"       ;;
    esac
}

print_separator() {
    echo "--------------------------------------------------------------------------------------------------------------------------------"
}

print_header() {
    echo ""
    echo "================================================================================================================================"
    echo "                           RDBMS StorageType Test Summary - All CSPs (MariaDB)"
    echo "================================================================================================================================"
    echo ""
    printf "%-12s | %-20s | %-18s | %-6s | %-14s | %-10s | %-s\n" \
        "CSP" "StorageType(Req)" "StorageType(Ret)" "Result" "DB Status" "Elapsed" "Reason"
    print_separator
}

# ── Banner ─────────────────────────────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#    CB-Spider RDBMS StorageType Test (MariaDB) - Starting All CSPs           #"
echo "################################################################################"
echo ""
echo "Spider URL   : ${SPIDER_URL}"
echo "Max wait     : ${MAX_WAIT_SEC}s per instance"
echo "Poll interval: ${POLL_INTERVAL}s"
echo "Auto delete  : ${AUTO_DELETE}"
echo "Base dir     : ${BASE_DIR}"
echo ""
echo "Target CSPs  : AWS, Alibaba, OpenStack, NHN (MariaDB support only)"
echo ""
echo "Launching all CSP StorageType tests in parallel..."
echo ""

CSP_ORDER="AWS ALIBABA OPENSTACK NHN"

# ── Launch all CSP scripts in background ─────────────────────────────────────
for csp in ${CSP_ORDER}; do
    script=$(csp_script "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} (log: ${log_file})"
    "${SCRIPT_DIR}/${script}" > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_csp_${csp}.txt"
done

echo ""
echo "[MAIN] All CSP tests launched. Waiting for completion..."
echo "[MAIN] Monitor: tail -f ${LOG_DIR}/log_<csp>.txt"
echo ""

# ── Wait for all CSP background jobs ─────────────────────────────────────────
for csp in ${CSP_ORDER}; do
    pid=$(cat "${LOG_DIR}/pid_csp_${csp}.txt" 2>/dev/null)
    if [[ -n "${pid}" ]]; then
        wait "${pid}"
        exit_code=$?
        if [[ ${exit_code} -eq 0 ]]; then
            echo "[MAIN] ${csp} completed"
        else
            echo "[MAIN] ${csp} finished with exit code ${exit_code}"
        fi
    fi
done

echo ""
echo "[MAIN] All CSP tests finished. Collecting results..."
echo ""

# ── Print result table ────────────────────────────────────────────────────────
print_header

total=0
pass_count=0
fail_count=0
skip_count=0

for csp in ${CSP_ORDER}; do
    csp_lower=$(to_lower "${csp}")
    csp_results=$(ls "${RESULT_DIR}"/result_${csp_lower}_*.txt 2>/dev/null | sort)

    if [[ -z "${csp_results}" ]]; then
        printf "%-12s | %-18s | %-18s | %-6s | %-14s | %-10s | %-s\n" \
            "${csp}" "-" "-" "N/A" "NO_RESULT" "-" "-"
        fail_count=$((fail_count + 1))
        continue
    fi

    while IFS= read -r result_file; do
        [[ -f "${result_file}" ]] || continue
        IFS='|' read -r r_csp r_req r_ret r_pass r_status r_elapsed r_reason \
            < "${result_file}"

        printf "%-12s | %-20s | %-18s | %-6s | %-14s | %-10s | %-s\n" \
            "${r_csp}" "${r_req}" "${r_ret}" "${r_pass}" "${r_status}" "${r_elapsed}" "${r_reason}"

        total=$((total + 1))
        case "${r_pass}" in
            PASS) pass_count=$((pass_count + 1)) ;;
            FAIL) fail_count=$((fail_count + 1)) ;;
            SKIP) skip_count=$((skip_count + 1)) ;;
        esac
    done <<< "${csp_results}"
done

print_separator
echo ""
echo "Total: ${total}  PASS: ${pass_count}  FAIL: ${fail_count}  SKIP: ${skip_count}"
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
echo "================================================================================================================================"
echo ""

if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo ""
    echo "################################################################################"
    echo "#                         Per-CSP Detailed Logs                               #"
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
