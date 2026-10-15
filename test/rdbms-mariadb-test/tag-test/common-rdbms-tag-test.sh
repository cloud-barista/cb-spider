#!/bin/bash

# CB-Spider RDBMS Tag Management Common Test Script
# Flow: AddTag -> ListTag -> GetTag -> AddTag(2nd) -> ListTag -> RemoveTag -> ListTag(verify) -> Write result
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   RDBMS_NAME      - RDBMS instance name to tag
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL      - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)
#
# Result file format (8 fields):
#   CSP|AddTag|ListTag|GetTag|AddTag2|RemoveTag|VerifyRemoved|Elapsed

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

TAG_KEY1="spider-rdbms-tag"
TAG_VAL1="rdbms-tag-value"
TAG_KEY2="spider-rdbms-tag2"
TAG_VAL2="rdbms-tag-value2"

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

mkdir -p "$(dirname "${RESULT_FILE}")"

r_add1="FAIL"
r_list="FAIL"
r_get="FAIL"
r_add2="FAIL"
r_remove="FAIL"
r_verify="FAIL"

check_result() {
    local label="$1"
    local resp="$2"
    local err
    err=$(echo "${resp}" | jq -r '.message // empty' 2>/dev/null)
    if [[ -n "${err}" ]]; then
        echo "[${CSP_NAME}] ERROR on ${label}: ${err}"
        elapsed_fmt=$(format_elapsed $(($(date +%s) - start_time)))
        echo "${CSP_NAME}|${r_add1}|${r_list}|${r_get}|${r_add2}|${r_remove}|${r_verify}|${elapsed_fmt}" \
            > "${RESULT_FILE}"
        exit 1
    fi
}

echo "[${CSP_NAME}] [${timestamp}] Starting RDBMS tag management test for '${RDBMS_NAME}'..."

# ── Pre-cleanup: remove leftover tags from a previous run (ignore errors) ────
for key in "${TAG_KEY1}" "${TAG_KEY2}"; do
    curl -u "${SPIDER_AUTH}" -sX DELETE \
      "${SPIDER_URL}/spider/tag/${key}" \
      -H 'Content-Type: application/json' \
      -d "{
        \"ConnectionName\": \"${CONNECTION_NAME}\",
        \"ReqInfo\": {\"ResourceType\": \"rdbms\", \"ResourceName\": \"${RDBMS_NAME}\"}
      }" > /dev/null 2>&1
done

# ── AddTag (1st) ──────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] AddTag: key='${TAG_KEY1}' value='${TAG_VAL1}'"

add1_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/tag" \
  -H 'Content-Type: application/json' \
  -d "{
    \"ConnectionName\": \"${CONNECTION_NAME}\",
    \"ReqInfo\": {
      \"ResourceType\": \"rdbms\",
      \"ResourceName\": \"${RDBMS_NAME}\",
      \"Tag\": {\"Key\": \"${TAG_KEY1}\", \"Value\": \"${TAG_VAL1}\"}
    }
  }" 2>&1)

check_result "AddTag(1)" "${add1_resp}"
returned_key=$(echo "${add1_resp}" | jq -r '.Key // empty' 2>/dev/null)
if [[ "${returned_key}" == "${TAG_KEY1}" ]]; then
    r_add1="PASS"
fi
echo "[${CSP_NAME}] AddTag(1): ${r_add1}"

# ── ListTag ───────────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] ListTag: verifying '${TAG_KEY1}' is present"

list_resp=$(curl -u "${SPIDER_AUTH}" -s \
  "${SPIDER_URL}/spider/tag?ConnectionName=${CONNECTION_NAME}&ResourceType=rdbms&ResourceName=${RDBMS_NAME}" 2>&1)

check_result "ListTag" "${list_resp}"
if echo "${list_resp}" | jq -e --arg k "${TAG_KEY1}" '.tag[]? | select(.Key == $k)' > /dev/null 2>&1; then
    r_list="PASS"
fi
tag_count=$(echo "${list_resp}" | jq -r '.tag | length // 0' 2>/dev/null)
echo "[${CSP_NAME}] ListTag: ${r_list} (${tag_count} tag(s))"

# ── GetTag ────────────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] GetTag: key='${TAG_KEY1}'"

get_resp=$(curl -u "${SPIDER_AUTH}" -s \
  "${SPIDER_URL}/spider/tag/${TAG_KEY1}?ConnectionName=${CONNECTION_NAME}&ResourceType=rdbms&ResourceName=${RDBMS_NAME}" 2>&1)

check_result "GetTag" "${get_resp}"
got_val=$(echo "${get_resp}" | jq -r '.Value // empty' 2>/dev/null)
if [[ "${got_val}" == "${TAG_VAL1}" ]]; then
    r_get="PASS"
fi
echo "[${CSP_NAME}] GetTag: ${r_get} (Value='${got_val}')"

# ── AddTag (2nd) ──────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] AddTag(2): key='${TAG_KEY2}' value='${TAG_VAL2}'"

add2_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/tag" \
  -H 'Content-Type: application/json' \
  -d "{
    \"ConnectionName\": \"${CONNECTION_NAME}\",
    \"ReqInfo\": {
      \"ResourceType\": \"rdbms\",
      \"ResourceName\": \"${RDBMS_NAME}\",
      \"Tag\": {\"Key\": \"${TAG_KEY2}\", \"Value\": \"${TAG_VAL2}\"}
    }
  }" 2>&1)

check_result "AddTag(2)" "${add2_resp}"
returned_key2=$(echo "${add2_resp}" | jq -r '.Key // empty' 2>/dev/null)
if [[ "${returned_key2}" == "${TAG_KEY2}" ]]; then
    r_add2="PASS"
fi
echo "[${CSP_NAME}] AddTag(2): ${r_add2}"

# ── RemoveTag ─────────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] RemoveTag: key='${TAG_KEY1}'"

rm1_resp=$(curl -u "${SPIDER_AUTH}" -sX DELETE \
  "${SPIDER_URL}/spider/tag/${TAG_KEY1}" \
  -H 'Content-Type: application/json' \
  -d "{
    \"ConnectionName\": \"${CONNECTION_NAME}\",
    \"ReqInfo\": {
      \"ResourceType\": \"rdbms\",
      \"ResourceName\": \"${RDBMS_NAME}\"
    }
  }" 2>&1)

check_result "RemoveTag(1)" "${rm1_resp}"
rm_result=$(echo "${rm1_resp}" | jq -r '.Result // empty' 2>/dev/null)
if [[ "${rm_result}" == "true" ]]; then
    r_remove="PASS"
fi
echo "[${CSP_NAME}] RemoveTag: ${r_remove}"

# ── Verify Removed (with retry for eventual consistency) ─────────────────────
echo "[${CSP_NAME}] ListTag: verifying '${TAG_KEY1}' is removed"

VERIFY_MAX_WAIT="${VERIFY_MAX_WAIT:-60}"
VERIFY_INTERVAL="${VERIFY_INTERVAL:-5}"
verify_elapsed=0
while true; do
    verify_resp=$(curl -u "${SPIDER_AUTH}" -s \
      "${SPIDER_URL}/spider/tag?ConnectionName=${CONNECTION_NAME}&ResourceType=rdbms&ResourceName=${RDBMS_NAME}" 2>&1)

    check_result "ListTag(verify)" "${verify_resp}"
    if ! echo "${verify_resp}" | jq -e --arg k "${TAG_KEY1}" '.tag[]? | select(.Key == $k)' > /dev/null 2>&1; then
        r_verify="PASS"
        break
    fi
    if [[ ${verify_elapsed} -ge ${VERIFY_MAX_WAIT} ]]; then
        break
    fi
    echo "[${CSP_NAME}] VerifyRemoved: tag still present, retrying in ${VERIFY_INTERVAL}s (${verify_elapsed}s elapsed)..."
    sleep "${VERIFY_INTERVAL}"
    verify_elapsed=$((verify_elapsed + VERIFY_INTERVAL))
done

remaining=$(echo "${verify_resp}" | jq -r '.tag | length // 0' 2>/dev/null)
echo "[${CSP_NAME}] VerifyRemoved: ${r_verify} (${remaining} tag(s) remaining)"

# ── Cleanup: RemoveTag (2nd) ──────────────────────────────────────────────────
echo "[${CSP_NAME}] Cleanup: removing '${TAG_KEY2}'"
curl -u "${SPIDER_AUTH}" -sX DELETE \
  "${SPIDER_URL}/spider/tag/${TAG_KEY2}" \
  -H 'Content-Type: application/json' \
  -d "{
    \"ConnectionName\": \"${CONNECTION_NAME}\",
    \"ReqInfo\": {
      \"ResourceType\": \"rdbms\",
      \"ResourceName\": \"${RDBMS_NAME}\"
    }
  }" > /dev/null 2>&1

# ── Write Result ──────────────────────────────────────────────────────────────
end_time=$(date +%s)
elapsed_fmt=$(format_elapsed $((end_time - start_time)))

overall="PASS"
for r in "${r_add1}" "${r_list}" "${r_get}" "${r_add2}" "${r_remove}" "${r_verify}"; do
    [[ "${r}" != "PASS" ]] && overall="FAIL" && break
done

echo "[${CSP_NAME}] Tag test ${overall} (elapsed: ${elapsed_fmt})"
echo "[${CSP_NAME}]   AddTag(1)=${r_add1} ListTag=${r_list} GetTag=${r_get} AddTag(2)=${r_add2} RemoveTag=${r_remove} VerifyRemoved=${r_verify}"

# Format: CSP|AddTag|ListTag|GetTag|AddTag2|RemoveTag|VerifyRemoved|Elapsed
echo "${CSP_NAME}|${r_add1}|${r_list}|${r_get}|${r_add2}|${r_remove}|${r_verify}|${elapsed_fmt}" \
  > "${RESULT_FILE}"
