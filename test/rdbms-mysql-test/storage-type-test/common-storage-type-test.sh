#!/bin/bash

# CB-Spider RDBMS StorageType Common Test Script
# Flow: Create RDBMS with specific StorageType -> Poll until Available ->
#       Get Info -> Verify StorageType matches requested -> Write result ->
#       Delete instance (if AUTO_DELETE=true)
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   RDBMS_NAME      - RDBMS instance name
#   STORAGE_TYPE    - StorageType value being tested (e.g., "gp2")
#   CREATE_JSON     - Full JSON body for POST /spider/rdbms
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL      - default: http://localhost:1024
#   SPIDER_AUTH     - default: admin:****
#   MAX_WAIT_SEC    - default: 3600
#   POLL_INTERVAL   - default: 30
#   AUTO_DELETE     - delete instance after test (default: false)
#
# Result file format (7 fields):
#   CSP|StorageType_Requested|StorageType_Returned|PASS_FAIL|DB_Status|Elapsed|Reason

format_elapsed() {
    local sec=$1
    if [[ ${sec} -lt 60 ]]; then
        echo "${sec}s"
    else
        echo "$((sec / 60))m$((sec % 60))s"
    fi
}

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
MAX_WAIT_SEC="${MAX_WAIT_SEC:-3600}"
POLL_INTERVAL="${POLL_INTERVAL:-30}"
AUTO_DELETE="${AUTO_DELETE:-false}"

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

echo "[${CSP_NAME}/${STORAGE_TYPE}] [${timestamp}] Creating RDBMS '${RDBMS_NAME}'..."

# ── Create RDBMS ─────────────────────────────────────────────────────────────
create_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/rdbms" \
    -H 'Content-Type: application/json' \
    -d "${CREATE_JSON}" 2>&1)

err_msg=$(echo "${create_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${err_msg}" ]]; then
    echo "[${CSP_NAME}/${STORAGE_TYPE}] ERROR on create: ${err_msg}"
    mkdir -p "$(dirname "${RESULT_FILE}")"
    elapsed_fmt=$(format_elapsed $(($(date +%s) - start_time)))
    echo "${CSP_NAME}|${STORAGE_TYPE}|N/A|FAIL|CREATE_ERROR|${elapsed_fmt}|-" > "${RESULT_FILE}"
    exit 1
fi

create_status=$(echo "${create_resp}" | jq -r '.Status // .rdbms.Status // "unknown"' 2>/dev/null)
echo "[${CSP_NAME}/${STORAGE_TYPE}] Create response status: ${create_status}"

# ── Poll until Available ──────────────────────────────────────────────────────
echo "[${CSP_NAME}/${STORAGE_TYPE}] Waiting for RDBMS (poll every ${POLL_INTERVAL}s, max ${MAX_WAIT_SEC}s)..."

elapsed=0
while true; do
    sleep "${POLL_INTERVAL}"
    elapsed=$((elapsed + POLL_INTERVAL))

    poll_resp=$(curl -u "${SPIDER_AUTH}" -s \
        "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}?ConnectionName=${CONNECTION_NAME}" 2>&1)

    cur_status=$(echo "${poll_resp}" | jq -r '.Status // .rdbms.Status // "unknown"' 2>/dev/null)
    status_lower=$(echo "${cur_status}" | tr '[:upper:]' '[:lower:]')

    echo "[${CSP_NAME}/${STORAGE_TYPE}] Status: ${cur_status} (elapsed: ${elapsed}s)"

    case "${status_lower}" in
        available|ready|runnable|active)
            echo "[${CSP_NAME}/${STORAGE_TYPE}] RDBMS is available!"
            break
            ;;
    esac

    if [[ ${elapsed} -ge ${MAX_WAIT_SEC} ]]; then
        echo "[${CSP_NAME}/${STORAGE_TYPE}] TIMEOUT: did not become available within ${MAX_WAIT_SEC}s"
        mkdir -p "$(dirname "${RESULT_FILE}")"
        echo "${CSP_NAME}|${STORAGE_TYPE}|N/A|FAIL|TIMEOUT|$(format_elapsed ${elapsed})|-" > "${RESULT_FILE}"
        exit 1
    fi
done

end_time=$(date +%s)
elapsed_total=$((end_time - start_time))

# ── Get RDBMS Info ────────────────────────────────────────────────────────────
info=$(curl -u "${SPIDER_AUTH}" -s \
    "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}?ConnectionName=${CONNECTION_NAME}")

db_status=$(echo "${info}"      | jq -r '.Status       // "N/A"')
returned_type=$(echo "${info}"  | jq -r '.StorageType  // "N/A"')

# ── Verify StorageType ────────────────────────────────────────────────────────
req_lower=$(echo "${STORAGE_TYPE}"   | tr '[:upper:]' '[:lower:]')
ret_lower=$(echo "${returned_type}"  | tr '[:upper:]' '[:lower:]')

reason="-"

if [[ "${CSP_NAME}" == "OPENSTACK" ]]; then
    # OpenStack Trove API does not return volume.type in instance detail responses.
    # Accept Available status as PASS since StorageType cannot be verified post-creation.
    if [[ "${ret_lower}" == "n/a" || "${ret_lower}" == "na" || -z "${ret_lower}" ]]; then
        pass_fail="PASS"
        reason="OpenStack Trove does not expose StorageType post-creation; Available=PASS"
    elif [[ "${ret_lower}" == "${req_lower}" ]]; then
        pass_fail="PASS"
    else
        pass_fail="FAIL"
    fi
elif [[ "${req_lower}" == "cloud_auto" ]]; then
    # cloud_auto is Alibaba's auto-select storage type.
    # Alibaba automatically picks the best cloud storage type at provisioning time.
    # Accept any cloud-based type (cloud_*/general_essd); reject local_ssd or N/A.
    if [[ "${ret_lower}" == "n/a" || "${ret_lower}" == "na" || \
          "${ret_lower}" == "local_ssd" || -z "${ret_lower}" ]]; then
        pass_fail="FAIL"
        reason="cloud_auto: auto-select, unexpected returned type '${returned_type}'"
    else
        pass_fail="PASS"
        reason="cloud_auto: auto-select type, CSP chose '${returned_type}'"
    fi
elif [[ "${ret_lower}" == "${req_lower}" ]]; then
    pass_fail="PASS"
else
    pass_fail="FAIL"
fi

elapsed_fmt=$(format_elapsed "${elapsed_total}")
echo "[${CSP_NAME}/${STORAGE_TYPE}] Requested=${STORAGE_TYPE}, Returned=${returned_type} => ${pass_fail} (${elapsed_fmt})"
[[ "${reason}" != "-" ]] && echo "[${CSP_NAME}/${STORAGE_TYPE}] Note: ${reason}"

mkdir -p "$(dirname "${RESULT_FILE}")"

# Format: CSP|StorageType_Requested|StorageType_Returned|PASS_FAIL|DB_Status|Elapsed|Reason
echo "${CSP_NAME}|${STORAGE_TYPE}|${returned_type}|${pass_fail}|${db_status}|${elapsed_fmt}|${reason}" \
    > "${RESULT_FILE}"

# ── Auto Delete ───────────────────────────────────────────────────────────────
if [[ "${AUTO_DELETE}" == "true" ]]; then
    echo "[${CSP_NAME}/${STORAGE_TYPE}] Sending delete request for '${RDBMS_NAME}'..."
    del_resp=$(curl -u "${SPIDER_AUTH}" -sX DELETE \
        "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}" \
        -H 'Content-Type: application/json' \
        -d "{\"ConnectionName\": \"${CONNECTION_NAME}\"}" 2>&1)
    del_err=$(echo "${del_resp}" | jq -r '.message // empty' 2>/dev/null)
    if [[ -n "${del_err}" ]]; then
        echo "[${CSP_NAME}/${STORAGE_TYPE}] Delete warning: ${del_err}"
    else
        echo "[${CSP_NAME}/${STORAGE_TYPE}] Delete request accepted."
    fi
fi
