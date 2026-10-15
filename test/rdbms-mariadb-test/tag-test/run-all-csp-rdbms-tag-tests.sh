#!/bin/bash

# CB-Spider RDBMS Tag Management Test Runner (MariaDB)
# Runs AddTag/ListTag/GetTag/RemoveTag tests for CSPs where RDBMSMetaInfo.SupportsTag=true
# AND MariaDB is supported.
# Supported CSPs: AWS, ALIBABA
# (Azure/GCP/IBM/NCP/Tencent do not support MariaDB; OpenStack/NHN do not support Tag.)
# Prerequisite: RDBMS instances must already be created (run run-all-csp-rdbms-tests.sh first).
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

export RESULT_DIR="/tmp/rdbms_tag_results_$$"
LOG_DIR="/tmp/rdbms_tag_logs_$$"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── Helpers ───────────────────────────────────────────────────────────────────
to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

csp_script() {
    case "$1" in
        AWS)     echo "aws-rdbms-tag-test.sh"     ;;
        ALIBABA) echo "alibaba-rdbms-tag-test.sh" ;;
    esac
}

print_separator() {
    printf '%115s\n' '' | tr ' ' '-'
}

print_header() {
    echo ""
    printf '%115s\n' '' | tr ' ' '='
    echo "          RDBMS TAG MANAGEMENT TEST SUMMARY (SupportsTag=true CSPs) - MariaDB"
    printf '%115s\n' '' | tr ' ' '='
    echo ""
    printf "%-12s | %-8s | %-8s | %-7s | %-8s | %-10s | %-13s | %-10s\n" \
        "CSP" "AddTag" "ListTag" "GetTag" "AddTag2" "RemoveTag" "VerifyRemoved" "Elapsed"
    print_separator
}

# ── Launch ────────────────────────────────────────────────────────────────────
echo ""
echo "################################################################################"
echo "#    CB-Spider RDBMS Tag Management Test (MariaDB) - SupportsTag=true CSPs    #"
echo "#                          AWS / ALIBABA                                      #"
echo "################################################################################"
echo ""
echo "Spider URL : ${SPIDER_URL}"
echo "Result dir : ${RESULT_DIR}"
echo "Log dir    : ${LOG_DIR}"
echo ""
echo "Launching tag management tests in parallel..."
echo ""

# Only CSPs where RDBMSMetaInfo.SupportsTag=true AND MariaDB is supported
CSP_ORDER="AWS ALIBABA"

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
        IFS='|' read -r r_csp r_add1 r_list r_get r_add2 r_remove r_verify r_elapsed \
            < "${result_file}"
    else
        r_csp="${csp}"
        r_add1="NO_RESULT"
        r_list="-"
        r_get="-"
        r_add2="-"
        r_remove="-"
        r_verify="-"
        r_elapsed="-"
    fi

    printf "%-12s | %-8s | %-8s | %-7s | %-8s | %-10s | %-13s | %-10s\n" \
        "${r_csp}" "${r_add1}" "${r_list}" "${r_get}" "${r_add2}" "${r_remove}" "${r_verify}" "${r_elapsed}"

    if [[ "${r_add1}" == "PASS" && "${r_list}" == "PASS" && "${r_get}" == "PASS" && \
          "${r_add2}" == "PASS" && "${r_remove}" == "PASS" && "${r_verify}" == "PASS" ]]; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
done

print_separator
echo ""
printf "Total: %d PASS, %d FAIL\n" "${pass_count}" "${fail_count}"
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
printf '%115s\n' '' | tr ' ' '='
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

[[ ${fail_count} -eq 0 ]]
