#!/bin/bash

# CB-Spider VM Default-PublicIP Test - Basic Resource Cleanup Runner for All CSPs
# Deletes the KeyPair -> SecurityGroup -> VPC/Subnet on all 10 CSPs in parallel.
# Run this AFTER the VM instances have been deleted (delete-all-csp-vm.sh).
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

export RESULT_DIR="/tmp/vm_publicip_network_cleanup_results_$$"
LOG_DIR="/tmp/vm_publicip_network_cleanup_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

# csp_config CSP -> "CONNECTION_NAME VPC_NAME SG_NAME KEYPAIR_NAME" (space-separated)
csp_config() {
    case "$1" in
        AWS)       echo "aws-config01 vpc-01 sg-01 keypair-01"                ;;
        AZURE)     echo "azure-koreacentral-config vpc-01 sg-01 keypair-01"   ;;
        GCP)       echo "gcp-iowa-config vpc-01 sg-01 keypair-01"             ;;
        ALIBABA)   echo "alibaba-beijing-config vpc-01 sg-01 keypair-01"      ;;
        TENCENT)   echo "tencent-beijing3-config vpc-01 sg-01 keypair-01"     ;;
        IBM)       echo "ibm-us-east-1-config vpc-01 sg-01 keypair-01"        ;;
        OPENSTACK) echo "openstack-config01 vpc-01 sg-01 keypair-01"          ;;
        NCP)       echo "ncp-korea1-config vpc-01 sg-01 keypair-01"           ;;
        NHN)       echo "nhn-korea-pangyo1-config vpc-01 sg-01 keypair-01"    ;;
        KT)        echo "kt-mokdong1-config vpc-01 sg-01 keypair-01"          ;;
    esac
}

print_separator() { echo "------------------------------------------------------------"; }

echo ""
echo "################################################################################"
echo "#     CB-Spider VM Default-PublicIP Test - Basic Resource Cleanup - All CSPs  #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Launching basic resource cleanup on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN KT"

for csp in ${CSP_ORDER}; do
    read -r conn vpc sg kp <<< "$(csp_config "${csp}")"
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} basic-resource cleanup (log: ${log_file})"
    (
        export CSP_NAME="${csp}"
        export CONNECTION_NAME="${conn}"
        export VPC_NAME="${vpc}"
        export SG_NAME="${sg}"
        export KEYPAIR_NAME="${kp}"
        export RESULT_FILE="${RESULT_DIR}/result_$(to_lower "${csp}").txt"
        "${SCRIPT_DIR}/common-network-cleanup.sh"
    ) > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_${csp}.txt"
done

echo ""
echo "[MAIN] All CSP cleanups launched. Waiting for completion..."
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
echo "[MAIN] All CSP cleanups finished. Collecting results..."
echo ""

echo "============================================================"
echo "  BASIC RESOURCE CLEANUP SUMMARY - ALL CSPs (KeyPair/SG/VPC)"
echo "============================================================"
echo ""
printf "%-12s | %-10s | %-10s | %-10s | %-10s\n" "CSP" "VPC" "SG" "KeyPair" "Elapsed"
print_separator

for csp in ${CSP_ORDER}; do
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    if [[ -f "${result_file}" ]]; then
        IFS='|' read -r r_csp r_vpc r_sg r_kp r_elapsed < "${result_file}"
    else
        r_csp="${csp}"
        r_vpc="NO_RESULT"
        r_sg="-"
        r_kp="-"
        r_elapsed="-"
    fi

    printf "%-12s | %-10s | %-10s | %-10s | %-10s\n" \
        "${r_csp}" "${r_vpc}" "${r_sg}" "${r_kp}" "${r_elapsed}"
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
