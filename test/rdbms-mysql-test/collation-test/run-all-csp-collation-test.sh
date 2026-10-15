#!/bin/bash

# CB-Spider RDBMS Collation Configuration Test Runner
# Verifies the Collation parameter on CreateDatabase actually takes effect on the database engine
# (not just that the API call succeeds) for every CSP where CB-Spider currently supports it.
# Prerequisite: RDBMS instances must already be created (run ../run-all-csp-rdbms-tests.sh first).
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

export RESULT_DIR="/tmp/rdbms_collation_results_$$"
LOG_DIR="/tmp/rdbms_collation_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-collation-test.sh"       ;;
        AZURE)     echo "azure-collation-test.sh"     ;;
        GCP)       echo "gcp-collation-test.sh"       ;;
        ALIBABA)   echo "alibaba-collation-test.sh"   ;;
        TENCENT)   echo "tencent-collation-test.sh"   ;;
        IBM)       echo "ibm-collation-test.sh"       ;;
        OPENSTACK) echo "openstack-collation-test.sh" ;;
        NCP)       echo "ncp-collation-test.sh"       ;;
        NHN)       echo "nhn-collation-test.sh"       ;;
    esac
}

print_separator() {
    printf '%116s\n' '' | tr ' ' '-'
}

print_header() {
    echo ""
    printf '%116s\n' '' | tr ' ' '='
    echo "                        RDBMS COLLATION CONFIGURATION TEST SUMMARY"
    printf '%116s\n' '' | tr ' ' '='
    echo ""
    printf "%-10s | %-9s | %-9s | %-9s | %-9s | %-9s | %-9s | %-9s\n" \
        "CSP" "CreateCI" "MetaCI" "FuncCI" "CreateBin" "MetaBin" "FuncBin" "Elapsed"
    print_separator
}

# ── Launch ────────────────────────────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#     CB-Spider RDBMS Collation Configuration Test - Starting All CSPs        #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Scope: all 9 RDBMS-capable CSPs -- see README.md for the mechanism each one uses."
echo "Tencent always exercises the SQL ALTER DATABASE follow-up here (its native API has no"
echo "collation field at all). NHN requires \"Direct Control\" to already be enabled on the test"
echo "instance (see nhn-collation-test.sh) -- without it, NHN fails with the engine's own"
echo "permission error."
echo ""
echo "For each CSP, two scenarios are tested (Charset fixed at utf8mb4, only Collation varies):"
echo "  CI  = utf8mb4_general_ci (case-insensitive)"
echo "  Bin = utf8mb4_bin (byte-exact, case-sensitive)"
echo "  CreateX - CreateDatabase with Charset=utf8mb4, Collation=X returns \"created\""
echo "  MetaX   - information_schema.SCHEMATA.DEFAULT_COLLATION_NAME equals X"
echo "  FuncX   - inserting \"TestWord\" and comparing against \"testword\" (lowercase) matches"
echo "            under CI and does not match under Bin, proving the collation request was"
echo "            really honored rather than silently defaulted"
echo ""
echo "Launching collation tests on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN"

for csp in ${CSP_ORDER}; do
    script=$(csp_script "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} (log: ${log_file})"
    "${SCRIPT_DIR}/${script}" > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_${csp}.txt"
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
        IFS='|' read -r r_csp r_cci r_mci r_fci r_cbin r_mbin r_fbin r_elapsed \
            < "${result_file}"
    else
        r_csp="${csp}"
        r_cci="NO_RESULT"; r_mci="-"; r_fci="-"
        r_cbin="-"; r_mbin="-"; r_fbin="-"
        r_elapsed="-"
    fi

    printf "%-10s | %-9s | %-9s | %-9s | %-9s | %-9s | %-9s | %-9s\n" \
        "${r_csp}" "${r_cci}" "${r_mci}" "${r_fci}" "${r_cbin}" "${r_mbin}" "${r_fbin}" "${r_elapsed}"

    if [[ "${r_cci}" == "PASS" && "${r_mci}" == "PASS" && "${r_fci}" == "PASS" && \
          "${r_cbin}" == "PASS" && "${r_mbin}" == "PASS" && "${r_fbin}" == "PASS" ]]; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
done

print_separator
echo ""
printf "Total: %d PASS, %d FAIL\n" "${pass_count}" "${fail_count}"
echo ""
echo "Note: NHN fails here if \"Direct Control\" isn't enabled on the test instance (NHN console) --"
echo "that's an environment precondition, not a CB-Spider bug. See nhn-collation-test.sh."
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
printf '%116s\n' '' | tr ' ' '='
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
