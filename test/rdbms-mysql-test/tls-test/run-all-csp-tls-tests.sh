#!/bin/bash

# CB-Spider RDBMS TLS Connection Test Runner for All CSPs
# Verifies the 4-scenario client TLS matrix against each CSP's real RDBMS endpoint.
# Prerequisite: RDBMS instances must already be created (run ../run-all-csp-rdbms-tests.sh first).
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

export RESULT_DIR="/tmp/rdbms_tls_results_$$"
LOG_DIR="/tmp/rdbms_tls_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-tls-test.sh"       ;;
        AZURE)     echo "azure-tls-test.sh"     ;;
        GCP)       echo "gcp-tls-test.sh"       ;;
        ALIBABA)   echo "alibaba-tls-test.sh"   ;;
        TENCENT)   echo "tencent-tls-test.sh"   ;;
        IBM)       echo "ibm-tls-test.sh"       ;;
        OPENSTACK) echo "openstack-tls-test.sh" ;;
        NCP)       echo "ncp-tls-test.sh"       ;;
        NHN)       echo "nhn-tls-test.sh"       ;;
    esac
}

print_separator() {
    printf '%118s\n' '' | tr ' ' '-'
}

print_header() {
    echo ""
    printf '%118s\n' '' | tr ' ' '='
    echo "                            RDBMS TLS CONNECTION TEST SUMMARY - ALL CSPs"
    printf '%118s\n' '' | tr ' ' '='
    echo ""
    printf "%-12s | %-10s | %-11s | %-8s | %-8s | %-8s | %-8s | %-10s\n" \
        "CSP" "ReqSecTLS" "TLSAvailable" "S1" "S2" "S3" "S4" "Elapsed"
    print_separator
}

# ── Launch ────────────────────────────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#      CB-Spider RDBMS TLS Connection Multi-CSP Test - Starting All CSPs      #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Scenarios tested per CSP (ordered weakest -> strongest security):"
echo "  S1: tls=false, no CA                        -> expect FAIL when ON, SUCCESS when OFF"
echo "      (always attempted, even when the instance has no TLS at all -- this is then the"
echo "       only meaningful scenario, and confirms plain connectivity works)"
echo "  S2: tls=skip-verify, no CA                  -> expect SUCCESS always (encrypted, zero verification)"
echo "  S3: VERIFY_CA equivalent (chain trust only, no hostname check) -> expect SUCCESS always"
echo "  S4: tls=true, CA registered                 -> expect SUCCESS always (strongest: chain + hostname)"
echo "  S2-S4 are N/A when TLSAvailable=OFF (the instance doesn't support TLS at all)."
echo ""
echo "Launching TLS connection tests on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN"

# Stagger launches: each tlsprobe run already spaces its own 5 connection attempts apart
# (-inter-attempt-delay) to avoid tripping a burst-rate limiter somewhere in the local
# network path, but 9 CSPs launched at the exact same instant re-creates that same burst
# across CSPs. A small stagger keeps their connection attempts from all landing in the same
# window without meaningfully lengthening the total run (they still finish in parallel).
LAUNCH_STAGGER_SEC="${LAUNCH_STAGGER_SEC:-8}"

for csp in ${CSP_ORDER}; do
    script=$(csp_script "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} (log: ${log_file})"
    "${SCRIPT_DIR}/${script}" > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_${csp}.txt"
    sleep "${LAUNCH_STAGGER_SEC}"
done

echo ""
echo "[MAIN] All tests launched. Waiting for completion..."
echo "[MAIN] Monitor progress: tail -f ${LOG_DIR}/log_<csp_lowercase>.txt"
echo ""

# ── Wait for all ──────────────────────────────────────────────────────────────
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
echo "[MAIN] All tests finished. Collecting results..."
echo ""

# ── Print result table ────────────────────────────────────────────────────────
print_header

pass_count=0
fail_count=0

for csp in ${CSP_ORDER}; do
    result_file="${RESULT_DIR}/result_$(to_lower "${csp}").txt"

    if [[ -f "${result_file}" ]]; then
        IFS='|' read -r r_csp r_rst r_tls r_s1 r_s2 r_s3 r_s4 r_elapsed \
            < "${result_file}"
    else
        r_csp="${csp}"
        r_rst="NO_RESULT"
        r_tls="-"
        r_s1="-"; r_s2="-"; r_s3="-"; r_s4="-"
        r_elapsed="-"
    fi

    printf "%-12s | %-10s | %-11s | %-8s | %-8s | %-8s | %-8s | %-10s\n" \
        "${r_csp}" "${r_rst}" "${r_tls}" "${r_s1}" "${r_s2}" "${r_s3}" "${r_s4}" "${r_elapsed}"

    # N/A is a legitimate outcome for S2-S4 when TLSAvailable=OFF (nothing to test), so it
    # counts as PASS as long as no scenario actually FAILed. r_rst == "N/A" is different: that
    # means the whole check aborted (e.g. GetRDBMS/GetRDBMSSecureTransport itself failed), so
    # nothing could be verified at all -- that's a real FAIL.
    if [[ "${r_rst}" == "N/A" ]]; then
        fail_count=$((fail_count + 1))
    elif [[ ( "${r_s1}" == "PASS" || "${r_s1}" == "N/A" ) && \
            ( "${r_s2}" == "PASS" || "${r_s2}" == "N/A" ) && \
            ( "${r_s3}" == "PASS" || "${r_s3}" == "N/A" ) && \
            ( "${r_s4}" == "PASS" || "${r_s4}" == "N/A" ) ]]; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
done

print_separator
echo ""
printf "Total: %d PASS, %d FAIL\n" "${pass_count}" "${fail_count}"
echo "(S2-S4 = N/A counts as PASS when TLSAvailable=OFF -- there's nothing to test, not a failure)"
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
printf '%118s\n' '' | tr ' ' '='
echo ""

if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo ""
    echo "################################################################################"
    echo "#                          Per-CSP Detailed Logs                              #"
    echo "################################################################################"
    for csp in ${CSP_ORDER}; do
        log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
        echo ""
        echo "────────────────────────────── ${csp} ──────────────────────────────"
        [[ -f "${log_file}" ]] && cat "${log_file}" || echo "(no log)"
    done
fi

# Propagate failure to caller so a nonzero FAIL count fails this step
[[ ${fail_count} -eq 0 ]]
