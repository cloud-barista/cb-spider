#!/bin/bash
# ---------------------------------------------------------------
# Quota API Test Script
#   1) ListQuotaServiceType - Retrieve service type list
#   2) GetQuotaInfo per service type - Call and verify results
#
# Usage:
#   ./quota-info-test.sh [ConnectionName] [ServerAddress]
#
# Examples:
#   ./quota-info-test.sh                                   # defaults: azure-koreacentral-config, localhost:1024
#   ./quota-info-test.sh azure-eastus-config               # test with eastus connection
#   ./quota-info-test.sh azure-eastus-config localhost:2048 # custom server address
# ---------------------------------------------------------------

CONN_NAME="${1:-azure-koreacentral-config}"
SERVER="${2:-localhost:1024}"
BASE_URL="http://${SERVER}/spider"

# Basic Auth (set SPIDER_USERNAME / SPIDER_PASSWORD env vars)
AUTH=""
if [[ -n "${SPIDER_USERNAME}" && -n "${SPIDER_PASSWORD}" ]]; then
    AUTH="-u ${SPIDER_USERNAME}:${SPIDER_PASSWORD}"
fi

echo "============================================================"
echo " Quota API Test"
echo " Connection : ${CONN_NAME}"
echo " Server     : ${SERVER}"
echo " Auth       : ${AUTH:+enabled}"
echo "============================================================"
echo ""

# ----------------------------
# Step 1: ListQuotaServiceType
# ----------------------------
echo "------------------------------------------------------------"
echo "[Step 1] ListQuotaServiceType"
echo "------------------------------------------------------------"
echo "GET ${BASE_URL}/quotaservicetype?ConnectionName=${CONN_NAME}"
echo ""

SERVICE_TYPES_RAW=$(curl -s ${AUTH} "${BASE_URL}/quotaservicetype?ConnectionName=${CONN_NAME}")
echo "${SERVICE_TYPES_RAW}" | python3 -m json.tool 2>/dev/null || echo "${SERVICE_TYPES_RAW}"
echo ""

# Extract service type array using jq or python3
if command -v jq &>/dev/null; then
    SERVICE_TYPES=$(echo "${SERVICE_TYPES_RAW}" | jq -r '.ServiceTypes[]' 2>/dev/null)
else
    SERVICE_TYPES=$(echo "${SERVICE_TYPES_RAW}" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for st in data.get('ServiceTypes', []):
    print(st)
" 2>/dev/null)
fi

if [[ -z "${SERVICE_TYPES}" ]]; then
    echo "[ERROR] Failed to retrieve service type list."
    exit 1
fi

TOTAL=$(echo "${SERVICE_TYPES}" | wc -l)
echo "Found ${TOTAL} service type(s)"
echo ""

# ----------------------------
# Step 2: GetQuotaInfo per service type
# ----------------------------
echo "============================================================"
echo "[Step 2] GetQuotaInfo per Service Type"
echo "============================================================"
echo ""

SUCCESS=0
FAIL=0
IDX=0

for ST in ${SERVICE_TYPES}; do
    IDX=$((IDX + 1))
    echo "------------------------------------------------------------"
    echo "[${IDX}/${TOTAL}] ServiceType: ${ST}"
    echo "------------------------------------------------------------"
    echo "GET ${BASE_URL}/quotainfo?ConnectionName=${CONN_NAME}&ServiceType=${ST}"

    RESP=$(curl -s ${AUTH} -w "\n%{http_code}" "${BASE_URL}/quotainfo?ConnectionName=${CONN_NAME}&ServiceType=${ST}")

    # Separate HTTP status code and response body
    HTTP_CODE=$(echo "${RESP}" | tail -1)
    BODY=$(echo "${RESP}" | sed '$d')

    if [[ "${HTTP_CODE}" == "200" ]]; then
        # Extract quota item count
        if command -v jq &>/dev/null; then
            COUNT=$(echo "${BODY}" | jq '.Quotas | length' 2>/dev/null)
        else
            COUNT=$(echo "${BODY}" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(len(data.get('Quotas', [])))
" 2>/dev/null)
        fi
        echo "  => OK (HTTP ${HTTP_CODE}) - ${COUNT} quota item(s)"
        SUCCESS=$((SUCCESS + 1))
    else
        echo "  => FAIL (HTTP ${HTTP_CODE})"
        echo "  => ${BODY}" | head -3
        FAIL=$((FAIL + 1))
    fi
    echo ""
done

# ----------------------------
# Summary
# ----------------------------
echo "============================================================"
echo " Test Result Summary"
echo "============================================================"
echo " Connection      : ${CONN_NAME}"
echo " Service Types   : ${TOTAL}"
echo " Success         : ${SUCCESS}"
echo " Failure         : ${FAIL}"
echo "============================================================"

if [[ ${FAIL} -gt 0 ]]; then
    exit 1
fi
