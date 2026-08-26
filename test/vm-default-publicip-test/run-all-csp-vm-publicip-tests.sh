#!/bin/bash

# CB-Spider VM Default-PublicIP Test Runner for All CSPs
# Runs VM create -> wait Running -> record PublicIP -> Suspend -> record
# PublicIP -> Resume -> record PublicIP on all 10 CSPs in parallel, then
# collects and displays a unified result table comparing the PublicIP before
# suspend and after resume.
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export MAX_WAIT_SEC="${MAX_WAIT_SEC:-1800}"                 # wait for Running after create
export POLL_INTERVAL="${POLL_INTERVAL:-15}"
export SUSPEND_MAX_WAIT_SEC="${SUSPEND_MAX_WAIT_SEC:-600}"  # wait for Suspended
export RESUME_MAX_WAIT_SEC="${RESUME_MAX_WAIT_SEC:-600}"    # wait for Running after resume

# Temp directories
export RESULT_DIR="/tmp/vm_publicip_results_$$"
LOG_DIR="/tmp/vm_publicip_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-vm-publicip-test.sh"       ;;
        AZURE)     echo "azure-vm-publicip-test.sh"     ;;
        GCP)       echo "gcp-vm-publicip-test.sh"       ;;
        ALIBABA)   echo "alibaba-vm-publicip-test.sh"   ;;
        TENCENT)   echo "tencent-vm-publicip-test.sh"   ;;
        IBM)       echo "ibm-vm-publicip-test.sh"       ;;
        OPENSTACK) echo "openstack-vm-publicip-test.sh" ;;
        NCP)       echo "ncp-vm-publicip-test.sh"       ;;
        NHN)       echo "nhn-vm-publicip-test.sh"       ;;
        KT)        echo "kt-vm-publicip-test.sh"        ;;
    esac
}

print_separator() {
    printf '%168s\n' '' | tr ' ' '-'
}

print_header() {
    echo ""
    printf '%168s\n' '' | tr ' ' '='
    echo "                  VM DEFAULT-PUBLICIP TEST SUMMARY - ALL CSPs (Create -> SSH -> Suspend -> Resume -> SSH)"
    printf '%168s\n' '' | tr ' ' '='
    echo ""
    printf "%-11s | %-6s | %-17s | %-17s | %-17s | %-16s | %-9s | %-9s | %-8s\n" \
        "CSP" "Result" "PublicIP(Initial)" "PublicIP(Suspend)" "PublicIP(Resume)" "Initial->Resume" "SSH(Init)" "SSH(Resume)" "Elapsed"
    print_separator
}

# ── Run all CSP tests in parallel ────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#     CB-Spider VM Default-PublicIP Multi-CSP Test - Starting All CSPs        #"
echo "################################################################################"
echo ""
echo "Spider URL          : ${SPIDER_URL}"
echo "Max wait (create)    : ${MAX_WAIT_SEC}s per CSP"
echo "Max wait (suspend)   : ${SUSPEND_MAX_WAIT_SEC}s per CSP"
echo "Max wait (resume)    : ${RESUME_MAX_WAIT_SEC}s per CSP"
echo "Poll interval        : ${POLL_INTERVAL}s"
echo "Result dir           : ${RESULT_DIR}"
echo "Log dir              : ${LOG_DIR}"
echo ""
echo "Launching VM publicip test on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN KT"

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
        IFS='|' read -r r_csp r_status r_ip_init r_ip_susp r_ip_resume r_comparison r_ssh_init r_ssh_resume r_elapsed \
            < "${result_file}"
    else
        r_csp="${csp}"
        r_status="NO_RESULT"
        r_ip_init="-"
        r_ip_susp="-"
        r_ip_resume="-"
        r_comparison="-"
        r_ssh_init="-"
        r_ssh_resume="-"
        r_elapsed="-"
    fi

    printf "%-11s | %-6s | %-17s | %-17s | %-17s | %-16s | %-9s | %-11s | %-8s\n" \
        "${r_csp}" "${r_status}" "${r_ip_init}" "${r_ip_susp}" "${r_ip_resume}" "${r_comparison}" "${r_ssh_init}" "${r_ssh_resume}" "${r_elapsed}"

    [[ "${r_status}" != "OK" ]] && fail_count=$((fail_count + 1))
done

print_separator
echo ""
echo "Failed : ${fail_count}"
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
echo "Legend : Initial->Resume column — SAME (same PublicIP kept across suspend/resume),"
echo "         CHANGED (different PublicIP after resume), CHANGED_TO_NONE (had an IP, lost it),"
echo "         NONE_BOTH (no PublicIP either before or after — e.g. private-subnet VM)."
echo "         SSH(Init)/SSH(Resume) — OK (ssh login succeeded), FAIL (login failed after retries),"
echo "         NO_IP (no PublicIP to connect to), NO_KEY (PrivateKey file missing, see KEY_DIR)."
echo ""
printf '%168s\n' '' | tr ' ' '='
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

# Propagate failure to caller (all_test.sh) so a non-OK status fails this step
[[ ${fail_count} -eq 0 ]]
