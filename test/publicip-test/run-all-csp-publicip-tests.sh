#!/bin/bash

# CB-Spider PublicIP Test Runner for All CSPs
# Runs Create→Get→List→[Associate→Disassociate]→Delete lifecycle on all 10 CSPs
# in parallel, then collects and displays a unified result summary table.
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

# Unique temp directories per run (avoid collision when run in parallel)
export RESULT_DIR="/tmp/publicip_results_$$"
LOG_DIR="/tmp/publicip_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-publicip-test.sh"       ;;
        AZURE)     echo "azure-publicip-test.sh"     ;;
        GCP)       echo "gcp-publicip-test.sh"       ;;
        ALIBABA)   echo "alibaba-publicip-test.sh"   ;;
        TENCENT)   echo "tencent-publicip-test.sh"   ;;
        IBM)       echo "ibm-publicip-test.sh"       ;;
        OPENSTACK) echo "openstack-publicip-test.sh" ;;
        NCP)       echo "ncp-publicip-test.sh"       ;;
        NHN)       echo "nhn-publicip-test.sh"       ;;
        KT)        echo "kt-publicip-test.sh"        ;;
    esac
}

SEP_WIDTH=117
print_separator() { printf '%*s\n' "${SEP_WIDTH}" '' | tr ' ' '-'; }
print_double()    { printf '%*s\n' "${SEP_WIDTH}" '' | tr ' ' '='; }

print_header() {
    echo ""
    print_double
    printf "%*s\n" $(( (SEP_WIDTH + 44) / 2 )) "CB-Spider PublicIP Lifecycle Test Summary — All CSPs"
    print_double
    echo ""
    printf "%-10s | %-8s | %-18s | %-7s | %-5s | %-9s | %-7s | %-8s | %-7s | %-9s\n" \
        "CSP" "Overall" "IP Address" "Create" "Get" "List" "Assoc" "Dissoc" "Delete" "Elapsed"
    print_separator
}

# ── Banner ────────────────────────────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#         CB-Spider PublicIP Multi-CSP Test — Starting All CSPs               #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Lifecycle  : Create → Get → List → [Associate → Disassociate] → Delete"
echo "             (Associate/Disassociate run only if NIC_NAME or VM_NAME is set)"
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN KT"

# ── Launch all CSP tests in parallel ─────────────────────────────────────────
echo "Launching PublicIP tests on all CSPs in parallel..."
echo ""
for csp in ${CSP_ORDER}; do
    script=$(csp_script "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} (log: ${log_file})"
    # Pass RESULT_DIR so each CSP script can build its RESULT_FILE path
    RESULT_DIR="${RESULT_DIR}" "${SCRIPT_DIR}/${script}" > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_${csp}.txt"
done

echo ""
echo "[MAIN] All CSP tests launched. Waiting for completion..."
echo "[MAIN] Monitor progress: tail -f ${LOG_DIR}/log_<csp_lowercase>.txt"
echo ""

# ── Wait for all background jobs ──────────────────────────────────────────────
for csp in ${CSP_ORDER}; do
    pid=$(cat "${LOG_DIR}/pid_${csp}.txt" 2>/dev/null)
    if [[ -n "${pid}" ]]; then
        wait "${pid}"
        exit_code=$?
        if [[ ${exit_code} -eq 0 ]]; then
            echo "[MAIN] ${csp} completed successfully"
        else
            echo "[MAIN] ${csp} finished with exit code ${exit_code} (see ${LOG_DIR}/log_$(to_lower "${csp}").txt)"
        fi
    fi
done

echo ""
echo "[MAIN] All CSP tests finished. Collecting results..."

# ── Print result table ────────────────────────────────────────────────────────
print_header

pass_count=0; fail_count=0

for csp in ${CSP_ORDER}; do
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    if [[ -f "${result_file}" ]]; then
        IFS='|' read -r r_csp r_overall r_ip r_create r_get r_list r_assoc r_dissoc r_delete r_elapsed \
            < "${result_file}"
    else
        r_csp="${csp}"; r_overall="NO_RESULT"; r_ip="-"
        r_create="-"; r_get="-"; r_list="-"; r_assoc="-"; r_dissoc="-"; r_delete="-"; r_elapsed="-"
    fi

    printf "%-10s | %-8s | %-18s | %-7s | %-5s | %-9s | %-7s | %-8s | %-7s | %-9s\n" \
        "${r_csp}" "${r_overall}" "${r_ip}" \
        "${r_create}" "${r_get}" "${r_list}" \
        "${r_assoc}" "${r_dissoc}" "${r_delete}" "${r_elapsed}"

    if [[ "${r_overall}" == "PASS" ]]; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
done

print_separator
printf "%-10s   %-8s\n" "Total" "PASS=${pass_count}  FAIL=${fail_count}"
print_double
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""

# ── Per-CSP full log dump (controlled by VERBOSE=1) ──────────────────────────
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
