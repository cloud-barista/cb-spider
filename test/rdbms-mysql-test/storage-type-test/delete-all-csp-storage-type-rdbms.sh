#!/bin/bash

# CB-Spider RDBMS StorageType Test - Delete All Instances
# For each CSP, fetches StorageTypeOptions from rdbmsmetainfo, derives the
# RDBMS names used during StorageType testing (cb-mysql-st-<type>), and deletes
# them in parallel. All CSPs run concurrently.
#
# Author: CB-Spider Team
# Note: Written for bash 3.2+ compatibility (macOS default shell)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMMON_DELETE="${SCRIPT_DIR}/../common-rdbms-delete.sh"

# ── Configuration ─────────────────────────────────────────────────────────────
export SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
export SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
export MAX_WAIT_SEC="${MAX_WAIT_SEC:-1800}"
export POLL_INTERVAL="${POLL_INTERVAL:-15}"

BASE_DIR="/tmp/st_del_$$"
RESULT_DIR="${BASE_DIR}/results"
LOG_DIR="${BASE_DIR}/logs"
mkdir -p "${RESULT_DIR}" "${LOG_DIR}"

# ── CSP connection config map ─────────────────────────────────────────────────
csp_connection() {
    case "$1" in
        AWS)       echo "aws-config01"                 ;;
        AZURE)     echo "azure-koreacentral-config"    ;;
        GCP)       echo "gcp-iowa-config"              ;;
        ALIBABA)   echo "alibaba-beijing-config"       ;;
        TENCENT)   echo "tencent-beijing6-config"      ;;
        IBM)       echo "ibm-us-east-1-config"         ;;
        OPENSTACK) echo "openstack-config01"           ;;
        NCP)       echo "ncp-korea1-config"            ;;
        NHN)       echo "nhn-korea-pangyo1-config"     ;;
    esac
}

to_lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

# Derive safe name (same logic as in per-CSP test scripts)
st_safe_name() {
    echo "$1" | tr '[:upper:]' '[:lower:]' \
        | sed 's/[^a-z0-9]/-/g' | sed 's/--*/-/g' | sed 's/^-//;s/-$//' | cut -c1-15
}

print_separator() {
    echo "----------------------------------------------------------------------"
}

# ── Banner ────────────────────────────────────────────────────────────────────
echo ""
echo "########################################################################"
echo "#      CB-Spider RDBMS StorageType Test - Delete All Instances         #"
echo "########################################################################"
echo ""
echo "Spider URL   : ${SPIDER_URL}"
echo "Max wait     : ${MAX_WAIT_SEC}s per instance"
echo "Poll interval: ${POLL_INTERVAL}s"
echo "Base dir     : ${BASE_DIR}"
echo ""

CSP_ORDER="AWS AZURE GCP ALIBABA TENCENT IBM OPENSTACK NCP NHN"

# ── GCP fixed instance suffixes (matches gcp-storage-type-test.sh TEST_CASES) ─
GCP_SUFFIXES="pdssd-n2 pdssd-ent pdhdd-ent hdb-c4a hdb-n4"

# ── Per-CSP delete function (runs in background subshell) ─────────────────────
run_csp_delete() {
    local csp="$1"
    local conn="$2"
    local csp_lower
    csp_lower=$(to_lower "${csp}")

    # GCP uses fixed instance names (not derived from metainfo StorageTypeOptions)
    if [[ "${csp}" == "GCP" ]]; then
        echo "[${csp}] Using fixed instance list for deletion..."
        for suffix in ${GCP_SUFFIXES}; do
            local rdbms_name="cb-mysql-st-${suffix}"
            local result_file="${RESULT_DIR}/result_${csp_lower}_${suffix}.txt"
            local log_file="${LOG_DIR}/log_${csp_lower}_${suffix}.txt"

            echo "[${csp}] Deleting '${rdbms_name}'..."
            (
                export CSP_NAME="${csp}"
                export CONNECTION_NAME="${conn}"
                export RDBMS_NAME="${rdbms_name}"
                export RESULT_FILE="${result_file}"
                exec "${COMMON_DELETE}"
            ) > "${log_file}" 2>&1 &

            echo $! > "${LOG_DIR}/pid_${csp_lower}_${suffix}.txt"
        done

        for pid_file in "${LOG_DIR}"/pid_${csp_lower}_*.txt; do
            [[ -f "${pid_file}" ]] || continue
            pid=$(cat "${pid_file}")
            wait "${pid}"
        done
        echo "[${csp}] All delete jobs completed."
        return 0
    fi

    echo "[${csp}] Fetching StorageTypeOptions from rdbmsmetainfo..."
    meta_resp=$(curl -u "${SPIDER_AUTH}" -sX GET \
        "${SPIDER_URL}/spider/rdbmsmetainfo?DBEngine=mysql&ConnectionName=${conn}" 2>&1)

    err_msg=$(echo "${meta_resp}" | jq -r '.message // empty' 2>/dev/null)
    if [[ -n "${err_msg}" ]]; then
        echo "[${csp}] ERROR fetching metainfo: ${err_msg}"
        echo "${csp}|META_ERROR|${err_msg}|-" \
            > "${RESULT_DIR}/result_${csp_lower}_meta_error.txt"
        return 1
    fi

    storage_types=$(echo "${meta_resp}" | jq -r '.StorageTypeOptions[]? // empty' 2>/dev/null)
    if [[ -z "${storage_types}" ]]; then
        echo "[${csp}] No StorageTypeOptions - nothing to delete"
        echo "${csp}|SKIP|no StorageTypeOptions|-" \
            > "${RESULT_DIR}/result_${csp_lower}_skip.txt"
        return 0
    fi

    echo "[${csp}] StorageTypeOptions: $(echo "${storage_types}" | tr '\n' ' ')"

    # Launch one delete job per StorageType in parallel
    while IFS= read -r storage_type; do
        [[ -z "${storage_type}" ]] && continue

        st_safe=$(st_safe_name "${storage_type}")
        rdbms_name="cb-mysql-st-${st_safe}"
        result_file="${RESULT_DIR}/result_${csp_lower}_${st_safe}.txt"
        log_file="${LOG_DIR}/log_${csp_lower}_${st_safe}.txt"

        echo "[${csp}] Deleting '${rdbms_name}' (StorageType=${storage_type})..."
        (
            export CSP_NAME="${csp}"
            export CONNECTION_NAME="${conn}"
            export RDBMS_NAME="${rdbms_name}"
            export RESULT_FILE="${result_file}"
            exec "${COMMON_DELETE}"
        ) > "${log_file}" 2>&1 &

        echo $! > "${LOG_DIR}/pid_${csp_lower}_${st_safe}.txt"
    done <<< "${storage_types}"

    # Wait for all delete jobs for this CSP
    for pid_file in "${LOG_DIR}"/pid_${csp_lower}_*.txt; do
        [[ -f "${pid_file}" ]] || continue
        pid=$(cat "${pid_file}")
        wait "${pid}"
    done
    echo "[${csp}] All delete jobs completed."
}

# ── Launch per-CSP delete in parallel ─────────────────────────────────────────
echo "Launching parallel deletion on all CSPs..."
echo ""

for csp in ${CSP_ORDER}; do
    conn=$(csp_connection "${csp}")
    log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
    echo "[MAIN] Starting ${csp} delete (log: ${log_file})"
    run_csp_delete "${csp}" "${conn}" > "${log_file}" 2>&1 &
    echo $! > "${LOG_DIR}/pid_csp_${csp}.txt"
done

echo ""
echo "[MAIN] All CSP delete jobs launched. Waiting for completion..."
echo "[MAIN] Monitor: tail -f ${LOG_DIR}/log_<csp>.txt"
echo ""

# ── Wait for all CSP jobs ─────────────────────────────────────────────────────
for csp in ${CSP_ORDER}; do
    pid=$(cat "${LOG_DIR}/pid_csp_${csp}.txt" 2>/dev/null)
    if [[ -n "${pid}" ]]; then
        wait "${pid}"
        exit_code=$?
        if [[ ${exit_code} -eq 0 ]]; then
            echo "[MAIN] ${csp} delete completed"
        else
            echo "[MAIN] ${csp} delete finished with exit code ${exit_code}"
        fi
    fi
done

echo ""
echo "[MAIN] All CSP delete jobs finished. Collecting results..."
echo ""

# ── Print result table ────────────────────────────────────────────────────────
echo "========================================================================"
echo "         RDBMS StorageType Test - DELETE SUMMARY - All CSPs"
echo "========================================================================"
echo ""
printf "%-12s | %-18s | %-14s | %-20s | %-10s\n" \
    "CSP" "RDBMS Name" "Result" "Detail" "Elapsed"
print_separator

total=0
deleted_count=0
skipped_count=0
error_count=0

for csp in ${CSP_ORDER}; do
    csp_lower=$(to_lower "${csp}")
    csp_results=$(ls "${RESULT_DIR}"/result_${csp_lower}_*.txt 2>/dev/null | sort)

    if [[ -z "${csp_results}" ]]; then
        printf "%-12s | %-18s | %-14s | %-20s | %-10s\n" \
            "${csp}" "-" "NO_RESULT" "-" "-"
        continue
    fi

    while IFS= read -r result_file; do
        [[ -f "${result_file}" ]] || continue
        IFS='|' read -r r_csp r_result r_detail r_elapsed < "${result_file}"

        # Derive RDBMS name from filename (result_<csp>_<st_safe>.txt)
        fname=$(basename "${result_file}" .txt)
        # Remove "result_<csp_lower>_" prefix to get st_safe
        st_safe="${fname#result_${csp_lower}_}"
        rdbms_name="cb-mysql-st-${st_safe}"
        [[ "${st_safe}" == "skip" || "${st_safe}" == "meta_error" ]] && rdbms_name="-"

        printf "%-12s | %-18s | %-14s | %-20s | %-10s\n" \
            "${r_csp}" "${rdbms_name}" "${r_result}" "${r_detail}" "${r_elapsed}"

        total=$((total + 1))
        case "${r_result}" in
            DELETED) deleted_count=$((deleted_count + 1)) ;;
            SKIP|NOT_FOUND) skipped_count=$((skipped_count + 1)) ;;
            *) error_count=$((error_count + 1)) ;;
        esac
    done <<< "${csp_results}"
done

print_separator
echo ""
echo "Total: ${total}  DELETED: ${deleted_count}  SKIPPED: ${skipped_count}  ERROR: ${error_count}"
echo ""
echo "Logs   : ${LOG_DIR}/"
echo "Results: ${RESULT_DIR}/"
echo ""
echo "========================================================================"
echo ""

# ── Per-CSP detailed logs (VERBOSE=1) ─────────────────────────────────────────
if [[ "${VERBOSE:-0}" == "1" ]]; then
    echo ""
    echo "########################################################################"
    echo "#                       Per-CSP Detailed Logs                         #"
    echo "########################################################################"
    for csp in ${CSP_ORDER}; do
        log_file="${LOG_DIR}/log_$(to_lower "${csp}").txt"
        echo ""
        echo "────────────────────────────── ${csp} ──────────────────────────────"
        [[ -f "${log_file}" ]] && cat "${log_file}" || echo "(no log)"
    done
fi
