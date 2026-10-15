#!/bin/bash

# CB-Spider VM Default-PublicIP Test - Basic Resource Prepare Runner for All CSPs
# Creates the VPC/Subnet -> SecurityGroup -> KeyPair required before running
# the VM publicip tests, on all 10 CSPs in parallel.
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

# Temp directories
export RESULT_DIR="/tmp/vm_publicip_network_results_$$"
LOG_DIR="/tmp/vm_publicip_network_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-network-prepare.sh"       ;;
        AZURE)     echo "azure-network-prepare.sh"     ;;
        GCP)       echo "gcp-network-prepare.sh"       ;;
        ALIBABA)   echo "alibaba-network-prepare.sh"   ;;
        TENCENT)   echo "tencent-network-prepare.sh"   ;;
        IBM)       echo "ibm-network-prepare.sh"       ;;
        OPENSTACK) echo "openstack-network-prepare.sh" ;;
        NCP)       echo "ncp-network-prepare.sh"       ;;
        NHN)       echo "nhn-network-prepare.sh"       ;;
        KT)        echo "kt-network-prepare.sh"        ;;
    esac
}

print_separator() {
    echo "------------------------------------------------------------"
}

# ── Run all CSP prepares in parallel ─────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#     CB-Spider VM Default-PublicIP Test - Basic Resource Prepare - All CSPs  #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Launching basic resource preparation on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN KT"

for csp in ${CSP_ORDER}; do
    script=$(csp_script "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} basic-resource prepare (log: ${log_file})"
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
echo "  BASIC RESOURCE PREPARE SUMMARY - ALL CSPs (VPC/SG/KeyPair)"
echo "============================================================"
echo ""
printf "%-12s | %-10s | %-10s | %-10s | %-10s\n" "CSP" "VPC" "SG" "KeyPair" "Elapsed"
print_separator

fail_count=0

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

# ── Per-CSP full log dump (optional, controlled by VERBOSE=1) ────────────────
if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo ""
    for csp in ${CSP_ORDER}; do
        log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
        echo "────────────── ${csp} ──────────────"
        [[ -f "${log_file}" ]] && cat "${log_file}" || echo "(no log)"
        echo ""
    done
fi

[[ ${fail_count} -eq 0 ]]
