#!/bin/bash

# CB-Spider RDBMS Delete Script for All CSPs
# Deletes RDBMS instances on all 9 CSPs in parallel and reports results.
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export MAX_WAIT_SEC="${MAX_WAIT_SEC:-1800}"  # 30 min timeout per CSP
export POLL_INTERVAL="${POLL_INTERVAL:-15}"  # poll every 15s

RDBMS_NAME="cb-spider-mysql-test"

RESULT_DIR="/tmp/rdbms_del_results_$$"
LOG_DIR="/tmp/rdbms_del_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── CSP connection config map ─────────────────────────────────────────────────
csp_connection() {
    case "$1" in
        AWS)       echo "aws-config01"                 ;;
        AZURE)     echo "azure-koreacentral-config"    ;;
        GCP)       echo "gcp-iowa-config"              ;;
        ALIBABA)   echo "alibaba-beijing-config"       ;;
        TENCENT)   echo "tencent-beijing3-config"      ;;
        IBM)       echo "ibm-us-east-1-config"         ;;
        OPENSTACK) echo "openstack-config01"           ;;
        NCP)       echo "ncp-korea1-config"            ;;
        NHN)       echo "nhn-korea-pangyo1-config"     ;;
    esac
}

to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

print_separator() {
    echo "------------------------------------------------------------"
}

# ── Launch ────────────────────────────────────────────────────────────────────
echo ""
echo "############################################################"
echo "#     CB-Spider RDBMS Delete - All CSPs                    #"
echo "############################################################"
echo ""
echo "RDBMS Name   : ${RDBMS_NAME}"
echo "Spider URL   : ${SPIDER_URL}"
echo "Max wait     : ${MAX_WAIT_SEC}s per CSP"
echo "Poll interval: ${POLL_INTERVAL}s"
echo ""
echo "Launching parallel deletion on all CSPs..."
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN"

for csp in ${CSP_ORDER}; do
    conn=$(csp_connection "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    echo "[MAIN] Deleting ${csp} (log: ${log_file})"

    (
        export CSP_NAME="${csp}"
        export CONNECTION_NAME="${conn}"
        export RDBMS_NAME="${RDBMS_NAME}"
        export RESULT_FILE="${result_file}"
        exec "${SCRIPT_DIR}/common-rdbms-delete.sh"
    ) > "${log_file}" 2>&1 &

    echo $! > "${LOG_DIR}/pid_${csp}.txt"
done

echo ""
echo "[MAIN] All deletions launched. Waiting for completion..."
echo "[MAIN] Monitor progress: tail -f ${LOG_DIR}/log_<csp_lowercase>.txt"
echo ""

# ── Wait for all ──────────────────────────────────────────────────────────────
for csp in ${CSP_ORDER}; do
    pid=$(cat "${LOG_DIR}/pid_${csp}.txt" 2>/dev/null)
    if [[ -n "${pid}" ]]; then
        wait "${pid}"
        exit_code=$?
        if [[ ${exit_code} -eq 0 ]]; then
            echo "[MAIN] ${csp} deletion completed"
        else
            echo "[MAIN] ${csp} deletion failed (exit: ${exit_code}, check ${LOG_DIR}/log_$(to_lower "${csp}").txt)"
        fi
    fi
done

echo ""
echo "[MAIN] All deletions finished. Collecting results..."
echo ""

# ── Result table ──────────────────────────────────────────────────────────────
echo "============================================================"
echo "         RDBMS DELETE SUMMARY - ALL CSPs"
echo "============================================================"
echo ""
printf "%-12s | %-14s | %-20s | %-10s\n" "CSP" "Result" "Detail" "Elapsed"
print_separator

for csp in ${CSP_ORDER}; do
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    if [[ -f "${result_file}" ]]; then
        IFS='|' read -r r_csp r_result r_detail r_elapsed < "${result_file}"
    else
        r_csp="${csp}"
        r_result="NO_RESULT"
        r_detail="-"
        r_elapsed="-"
    fi

    printf "%-12s | %-14s | %-20s | %-10s\n" \
        "${r_csp}" "${r_result}" "${r_detail}" "${r_elapsed}"
done

print_separator
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
echo "============================================================"
echo ""

# ── Per-CSP log dump (VERBOSE=1) ─────────────────────────────────────────────
if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo ""
    for csp in ${CSP_ORDER}; do
        log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
        echo "────────────── ${csp} ──────────────"
        [[ -f "${log_file}" ]] && cat "${log_file}" || echo "(no log)"
        echo ""
    done
fi
