#!/bin/bash

# CB-Spider Network Prerequisite Prepare Runner for All CSPs (MariaDB Test)
# Creates the VPC/Subnet (and Security Group, for AWS) required before
# running the RDBMS creation tests, on the CSPs that support MariaDB
# (AWS, Alibaba, OpenStack, NHN) in parallel.
# Azure/GCP/IBM/NCP/Tencent do not support MariaDB and are excluded so their
# VPCs are never created and their errors don't clutter the results.
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

# Temp directories
export RESULT_DIR="/tmp/rdbms_network_results_$$"
LOG_DIR="/tmp/rdbms_network_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-network-prepare.sh"       ;;
        ALIBABA)   echo "alibaba-network-prepare.sh"   ;;
        OPENSTACK) echo "openstack-network-prepare.sh" ;;
        NHN)       echo "nhn-network-prepare.sh"       ;;
    esac
}

print_separator() {
    echo "------------------------------------------------------------"
}

# ── Run all CSP prepares in parallel ─────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#     CB-Spider Network Prerequisite Prepare - All CSPs (MariaDB Test)        #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Launching network preparation on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS ALIBABA OPENSTACK NHN"

for csp in ${CSP_ORDER}; do
    script=$(csp_script "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} network prepare (log: ${log_file})"
    "${SCRIPT_DIR}/${script}" > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_${csp}.txt"
done

echo ""
echo "[MAIN] All CSP preparations launched. Waiting for completion..."
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
echo "[MAIN] All CSP preparations finished. Collecting results..."
echo ""

# ── Print result table ────────────────────────────────────────────────────────
echo "============================================================"
echo "     NETWORK PREREQUISITE PREPARE SUMMARY - ALL CSPs"
echo "============================================================"
echo ""
printf "%-12s | %-10s | %-10s | %-10s\n" "CSP" "VPC" "SG" "Elapsed"
print_separator

fail_count=0

for csp in ${CSP_ORDER}; do
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    if [[ -f "${result_file}" ]]; then
        IFS='|' read -r r_csp r_vpc r_sg r_elapsed < "${result_file}"
    else
        r_csp="${csp}"
        r_vpc="NO_RESULT"
        r_sg="-"
        r_elapsed="-"
        fail_count=$((fail_count + 1))
    fi

    printf "%-12s | %-10s | %-10s | %-10s\n" "${r_csp}" "${r_vpc}" "${r_sg}" "${r_elapsed}"

    case "${r_vpc}" in
        CREATED|EXISTS) ;;
        *) fail_count=$((fail_count + 1)) ;;
    esac
done

print_separator
echo ""
echo "Failed : ${fail_count}"
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
echo "============================================================"
echo ""

[[ ${fail_count} -eq 0 ]]
