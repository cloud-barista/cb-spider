#!/bin/bash

# CB-Spider VM Default-PublicIP Test - VM Delete Runner for All CSPs
# Terminates the VM instance on all 10 CSPs in parallel and waits for removal.
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export MAX_WAIT_SEC="${MAX_WAIT_SEC:-900}"
export POLL_INTERVAL="${POLL_INTERVAL:-15}"

export RESULT_DIR="/tmp/vm_publicip_delete_results_$$"
LOG_DIR="/tmp/vm_publicip_delete_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_connection() {
    case "$1" in
        AWS)       echo "aws-config01"                ;;
        AZURE)     echo "azure-koreacentral-config"    ;;
        GCP)       echo "gcp-iowa-config"               ;;
        ALIBABA)   echo "alibaba-beijing-config"        ;;
        TENCENT)   echo "tencent-beijing3-config"       ;;
        IBM)       echo "ibm-us-east-1-config"          ;;
        OPENSTACK) echo "openstack-config01"            ;;
        NCP)       echo "ncp-korea1-config"              ;;
        NHN)       echo "nhn-korea-pangyo1-config"       ;;
        KT)        echo "kt-mokdong1-config"             ;;
    esac
}

print_separator() { echo "------------------------------------------------------------"; }

echo ""
echo "################################################################################"
echo "#      CB-Spider VM Default-PublicIP Test - VM Delete - All CSPs              #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Launching VM deletion on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN KT"

for csp in ${CSP_ORDER}; do
    conn=$(csp_connection "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} VM delete (log: ${log_file})"
    (
        export CSP_NAME="${csp}"
        export CONNECTION_NAME="${conn}"
        export VM_NAME="cb-spider-publicip-test"
        export RESULT_FILE="${RESULT_DIR}/result_$(to_lower "${csp}").txt"
        "${SCRIPT_DIR}/common-vm-delete.sh"
    ) > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_${csp}.txt"
done

echo ""
echo "[MAIN] All CSP VM deletions launched. Waiting for completion..."
echo ""

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
echo "[MAIN] All CSP VM deletions finished. Collecting results..."
echo ""

echo "============================================================"
echo "          VM DELETE SUMMARY - ALL CSPs"
echo "============================================================"
echo ""
printf "%-12s | %-15s | %-20s | %-10s\n" "CSP" "Result" "Detail" "Elapsed"
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

    printf "%-12s | %-15s | %-20s | %-10s\n" "${r_csp}" "${r_result}" "${r_detail}" "${r_elapsed}"
done

print_separator
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
echo "============================================================"
echo ""

if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo ""
    for csp in ${CSP_ORDER}; do
        log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
        echo "────────────── ${csp} ──────────────"
        [[ -f "${log_file}" ]] && cat "${log_file}" || echo "(no log)"
        echo ""
    done
fi
