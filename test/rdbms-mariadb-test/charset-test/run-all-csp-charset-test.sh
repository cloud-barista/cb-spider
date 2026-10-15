#!/bin/bash

# CB-Spider RDBMS Charset Configuration Test Runner (MariaDB engine)
# Verifies the Charset parameter on CreateDatabase actually takes effect on the database engine
# (not just that the API call succeeds) for every CSP where CB-Spider currently supports it on
# MariaDB.
# Prerequisite: RDBMS instances must already be created (run ../run-all-csp-rdbms-tests.sh first).
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

export RESULT_DIR="/tmp/rdbms_charset_results_$$"
LOG_DIR="/tmp/rdbms_charset_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)       echo "aws-charset-test.sh"       ;;
        ALIBABA)   echo "alibaba-charset-test.sh"   ;;
        OPENSTACK) echo "openstack-charset-test.sh" ;;
        NHN)       echo "nhn-charset-test.sh"       ;;
    esac
}

print_separator() {
    printf '%116s\n' '' | tr ' ' '-'
}

print_header() {
    echo ""
    printf '%116s\n' '' | tr ' ' '='
    echo "                    RDBMS CHARSET CONFIGURATION TEST SUMMARY (MariaDB)"
    printf '%116s\n' '' | tr ' ' '='
    echo ""
    printf "%-10s | %-9s | %-9s | %-9s | %-9s | %-9s | %-9s | %-9s\n" \
        "CSP" "CreateU8" "MetaU8" "FuncU8" "CreateL1" "MetaL1" "FuncL1" "Elapsed"
    print_separator
}

# ── Launch ────────────────────────────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#   CB-Spider RDBMS Charset Configuration Test (MariaDB) - Starting All CSPs  #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Scope: AWS, Alibaba, OpenStack, NHN -- CSPs that (a) support MariaDB at all and (b) have the"
echo "Charset parameter wired up in CB-Spider today. Azure/GCP/IBM don't support MariaDB as an"
echo "engine at all; Tencent/NCP support neither. See README.md for the full matrix."
echo ""
echo "NHN requires \"Direct Control\" to already be enabled on the test instance (see"
echo "nhn-charset-test.sh) -- without it, NHN fails here with the engine's own permission error."
echo ""
echo "For each CSP, two scenarios are tested (U8 = utf8mb4, L1 = latin1):"
echo "  CreateX - CreateDatabase with Charset=X returns \"created\""
echo "  MetaX   - information_schema.SCHEMATA.DEFAULT_CHARACTER_SET_NAME equals X"
echo "  FuncX   - functional INSERT/SELECT round-trip of a Korean+emoji string behaves as"
echo "            expected for X (exact match for utf8mb4; rejected or mangled -- never an"
echo "            exact match -- for latin1, proving the charset request was really honored)"
echo ""
echo "Launching charset tests on all CSPs in parallel..."
echo ""

CSP_ORDER="AWS ALIBABA OPENSTACK NHN"

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
        IFS='|' read -r r_csp r_cu8 r_mu8 r_fu8 r_cl1 r_ml1 r_fl1 r_elapsed \
            < "${result_file}"
    else
        r_csp="${csp}"
        r_cu8="NO_RESULT"; r_mu8="-"; r_fu8="-"
        r_cl1="-"; r_ml1="-"; r_fl1="-"
        r_elapsed="-"
    fi

    printf "%-10s | %-9s | %-9s | %-9s | %-9s | %-9s | %-9s | %-9s\n" \
        "${r_csp}" "${r_cu8}" "${r_mu8}" "${r_fu8}" "${r_cl1}" "${r_ml1}" "${r_fl1}" "${r_elapsed}"

    if [[ "${r_cu8}" == "PASS" && "${r_mu8}" == "PASS" && "${r_fu8}" == "PASS" && \
          "${r_cl1}" == "PASS" && "${r_ml1}" == "PASS" && "${r_fl1}" == "PASS" ]]; then
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
echo "that's an environment precondition, not a CB-Spider bug. See nhn-charset-test.sh."
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
