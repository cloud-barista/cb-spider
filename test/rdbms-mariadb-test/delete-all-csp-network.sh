#!/bin/bash

# CB-Spider Network Prerequisite Cleanup Script for All CSPs (MariaDB Test)
# Deletes the VPC/Subnet (and Security Group, for AWS) created by
# run-all-csp-network-prepare.sh, on the CSPs that support MariaDB (AWS,
# Alibaba, OpenStack, NHN) in parallel.
# Azure/GCP/IBM/NCP/Tencent do not support MariaDB and are excluded, so their
# VPCs are never created in the first place.
# Run this AFTER delete-all-csp-rdbms.sh, since the RDBMS instance must be
# gone before its VPC/Subnet/SecurityGroup can be removed.
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

VPC_NAME="vpc-01"

RESULT_DIR="/tmp/rdbms_network_del_results_$$"
LOG_DIR="/tmp/rdbms_network_del_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── CSP connection config / SG map ────────────────────────────────────────────
csp_connection() {
    case "$1" in
        AWS)       echo "aws-config01"                 ;;
        ALIBABA)   echo "alibaba-beijing-config"       ;;
        OPENSTACK) echo "openstack-config01"           ;;
        NHN)       echo "nhn-korea-pangyo1-config"     ;;
    esac
}

# Only AWS has a Security Group among this test suite's prerequisites
csp_sg_name() {
    case "$1" in
        AWS) echo "sg-01" ;;
        *)   echo ""      ;;
    esac
}

to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

print_separator() {
    echo "------------------------------------------------------------"
}

# ── Launch ────────────────────────────────────────────────────────────────────
echo ""
echo "############################################################"
echo "#  CB-Spider Network Prerequisite Cleanup - All CSPs       #"
echo "#                   (MariaDB Test)                         #"
echo "############################################################"
echo ""
echo "VPC Name   : ${VPC_NAME}"
echo "Spider URL : ${SPIDER_URL}"
echo ""
echo "Launching parallel cleanup on all CSPs..."
echo ""

CSP_ORDER="AWS ALIBABA OPENSTACK NHN"

for csp in ${CSP_ORDER}; do
    conn=$(csp_connection "${csp}")
    sg_name=$(csp_sg_name "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    echo "[MAIN] Cleaning up ${csp} (log: ${log_file})"

    (
        export CSP_NAME="${csp}"
        export CONNECTION_NAME="${conn}"
        export VPC_NAME="${VPC_NAME}"
        export SG_NAME="${sg_name}"
        export RESULT_FILE="${result_file}"
        exec "${SCRIPT_DIR}/common-network-cleanup.sh"
    ) > "${log_file}" 2>&1 &

    echo $! > "${LOG_DIR}/pid_${csp}.txt"
done

echo ""
echo "[MAIN] All cleanups launched. Waiting for completion..."
echo ""

# ── Wait for all ──────────────────────────────────────────────────────────────
for csp in ${CSP_ORDER}; do
    pid=$(cat "${LOG_DIR}/pid_${csp}.txt" 2>/dev/null)
    if [[ -n "${pid}" ]]; then
        wait "${pid}"
        exit_code=$?
        if [[ ${exit_code} -eq 0 ]]; then
            echo "[MAIN] ${csp} cleanup completed"
        else
            echo "[MAIN] ${csp} cleanup failed (exit: ${exit_code}, check ${LOG_DIR}/log_$(to_lower "${csp}").txt)"
        fi
    fi
done

echo ""
echo "[MAIN] All cleanups finished. Collecting results..."
echo ""

# ── Result table ──────────────────────────────────────────────────────────────
echo "============================================================"
echo "       NETWORK CLEANUP SUMMARY - ALL CSPs"
echo "============================================================"
echo ""
printf "%-12s | %-16s | %-12s | %-10s\n" "CSP" "VPC" "SG" "Elapsed"
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

    printf "%-12s | %-16s | %-12s | %-10s\n" "${r_csp}" "${r_vpc}" "${r_sg}" "${r_elapsed}"

    case "${r_vpc}" in
        DELETED|NOT_FOUND_OR_ERROR) ;;
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
