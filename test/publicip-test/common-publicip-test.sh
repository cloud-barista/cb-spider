#!/bin/bash

# CB-Spider PublicIP Common Test Script
# Flow: Create → Get → List → [Associate → Disassociate] → Delete → Write result
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   PUBLICIP_NAME   - PublicIP name to create
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   NIC_NAME        - NIC name for Associate/Disassociate test (skip if empty)
#   VM_NAME         - VM name for Associate test (NCP-style, skip if empty)
#   PRIVATE_IP      - Private IP for Associate (empty = auto/primary)
#   SPIDER_URL      - CB-Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)

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

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

r_create="-"; r_get="-"; r_list="-"; r_assoc="-"; r_dissoc="-"; r_delete="-"
ip_addr="-"

echo "[${CSP_NAME}] [${timestamp}] === PublicIP lifecycle test start ==="

# ── 1. Create ─────────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] [1] Creating PublicIP '${PUBLICIP_NAME}'..."
create_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/publicip" \
  -H 'Content-Type: application/json' \
  -d "{\"ConnectionName\":\"${CONNECTION_NAME}\",\"ReqInfo\":{\"Name\":\"${PUBLICIP_NAME}\",\"TagList\":[{\"Key\":\"env\",\"Value\":\"spider-test\"}]}}" 2>&1)

err_msg=$(echo "${create_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${err_msg}" ]]; then
    echo "[${CSP_NAME}] ERROR on Create: ${err_msg}"
    end_time=$(date +%s)
    elapsed_fmt=$(format_elapsed $((end_time - start_time)))
    mkdir -p "$(dirname "${RESULT_FILE}")"
    echo "${CSP_NAME}|CREATE_ERROR|${err_msg}|FAIL|-|-|-|-|-|${elapsed_fmt}" > "${RESULT_FILE}"
    exit 1
fi

ip_addr=$(echo "${create_resp}" | jq -r '.PublicIPAddress // "N/A"' 2>/dev/null)
create_status=$(echo "${create_resp}" | jq -r '.Status // "N/A"' 2>/dev/null)
echo "[${CSP_NAME}]   → IP=${ip_addr}, Status=${create_status}"
r_create="OK"

# ── 2. Get ────────────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] [2] Getting PublicIP '${PUBLICIP_NAME}'..."
get_resp=$(curl -u "${SPIDER_AUTH}" -sX GET "${SPIDER_URL}/spider/publicip/${PUBLICIP_NAME}" \
  -H 'Content-Type: application/json' \
  -d "{\"ConnectionName\":\"${CONNECTION_NAME}\"}" 2>&1)

get_name=$(echo "${get_resp}" | jq -r '.IId.NameId // empty' 2>/dev/null)
if [[ -n "${get_name}" ]]; then
    r_get="OK"
    echo "[${CSP_NAME}]   → Get OK"
else
    r_get="FAIL"
    echo "[${CSP_NAME}]   → Get FAILED: $(echo "${get_resp}" | jq -r '.message // "unknown"' 2>/dev/null)"
fi

# ── 3. List ───────────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] [3] Listing PublicIPs..."
list_resp=$(curl -u "${SPIDER_AUTH}" -sX GET "${SPIDER_URL}/spider/publicip" \
  -H 'Content-Type: application/json' \
  -d "{\"ConnectionName\":\"${CONNECTION_NAME}\"}" 2>&1)

list_count=$(echo "${list_resp}" | jq -r '.publicip | length' 2>/dev/null)
if [[ "${list_count:-0}" -gt 0 ]]; then
    r_list="OK(${list_count})"
    echo "[${CSP_NAME}]   → List OK: ${list_count} item(s)"
else
    r_list="FAIL"
    echo "[${CSP_NAME}]   → List FAILED or empty"
fi

# ── 4. Associate (optional — only if NIC_NAME or VM_NAME is set) ──────────────
if [[ -n "${NIC_NAME}" || -n "${VM_NAME}" ]]; then
    echo "[${CSP_NAME}] [4] Associating PublicIP..."
    if [[ -n "${NIC_NAME}" ]]; then
        assoc_body="{\"ConnectionName\":\"${CONNECTION_NAME}\",\"ReqInfo\":{\"NICName\":\"${NIC_NAME}\",\"PrivateIP\":\"${PRIVATE_IP:-}\"}}"
    else
        assoc_body="{\"ConnectionName\":\"${CONNECTION_NAME}\",\"ReqInfo\":{\"VMName\":\"${VM_NAME}\"}}"
    fi

    assoc_resp=$(curl -u "${SPIDER_AUTH}" -sX PUT "${SPIDER_URL}/spider/publicip/${PUBLICIP_NAME}/associate" \
      -H 'Content-Type: application/json' \
      -d "${assoc_body}" 2>&1)

    assoc_status=$(echo "${assoc_resp}" | jq -r '.Status // empty' 2>/dev/null)
    if [[ "${assoc_status}" == "Associated" ]]; then
        r_assoc="OK"
        echo "[${CSP_NAME}]   → Associate OK"
    else
        r_assoc="FAIL"
        echo "[${CSP_NAME}]   → Associate FAILED: $(echo "${assoc_resp}" | jq -r '.message // "unknown"' 2>/dev/null)"
    fi

    # ── 5. Disassociate ────────────────────────────────────────────────────────
    echo "[${CSP_NAME}] [5] Disassociating PublicIP..."
    dissoc_resp=$(curl -u "${SPIDER_AUTH}" -sX PUT "${SPIDER_URL}/spider/publicip/${PUBLICIP_NAME}/disassociate" \
      -H 'Content-Type: application/json' \
      -d "{\"ConnectionName\":\"${CONNECTION_NAME}\"}" 2>&1)

    dissoc_result=$(echo "${dissoc_resp}" | jq -r '.Result // empty' 2>/dev/null)
    if [[ "${dissoc_result}" == "true" ]]; then
        r_dissoc="OK"
        echo "[${CSP_NAME}]   → Disassociate OK"
    else
        r_dissoc="FAIL"
        echo "[${CSP_NAME}]   → Disassociate FAILED: $(echo "${dissoc_resp}" | jq -r '.message // "unknown"' 2>/dev/null)"
    fi
fi

# ── 6. Delete ─────────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] [6] Deleting PublicIP '${PUBLICIP_NAME}'..."
delete_resp=$(curl -u "${SPIDER_AUTH}" -sX DELETE "${SPIDER_URL}/spider/publicip/${PUBLICIP_NAME}" \
  -H 'Content-Type: application/json' \
  -d "{\"ConnectionName\":\"${CONNECTION_NAME}\"}" 2>&1)

delete_result=$(echo "${delete_resp}" | jq -r '.Result // empty' 2>/dev/null)
if [[ "${delete_result}" == "true" ]]; then
    r_delete="OK"
    echo "[${CSP_NAME}]   → Delete OK"
else
    r_delete="FAIL"
    echo "[${CSP_NAME}]   → Delete FAILED: $(echo "${delete_resp}" | jq -r '.message // "unknown"' 2>/dev/null)"
fi

end_time=$(date +%s)
elapsed_fmt=$(format_elapsed $((end_time - start_time)))
echo "[${CSP_NAME}] === Test complete (elapsed: ${elapsed_fmt}) ==="

# Overall: PASS only if Create, Get, List, Delete all OK
if [[ "${r_create}" == "OK" && "${r_get}" == "OK" && "${r_delete}" == "OK" ]]; then
    overall="PASS"
else
    overall="FAIL"
fi

mkdir -p "$(dirname "${RESULT_FILE}")"
# Format: CSP|Overall|IPAddress|Create|Get|List|Assoc|Dissoc|Delete|Elapsed
echo "${CSP_NAME}|${overall}|${ip_addr}|${r_create}|${r_get}|${r_list}|${r_assoc}|${r_dissoc}|${r_delete}|${elapsed_fmt}" \
  > "${RESULT_FILE}"
