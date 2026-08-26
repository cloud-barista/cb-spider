#!/bin/bash

# CB-Spider VM Default-PublicIP Test - Network/Basic Resource Cleanup Script
# Deletes the KeyPair -> SecurityGroup -> VPC/Subnet created by
# common-network-prepare.sh. Safe to run even if the resources don't exist.
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   VPC_NAME        - VPC name to delete
#   SG_NAME         - Security Group name to delete
#   KEYPAIR_NAME    - KeyPair name to delete
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL      - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

# CSPs release the network resources (ENI/IP/subnet association) that were held
# by the just-deleted VM asynchronously, so a delete attempted right after VM
# deletion can hit a transient DependencyViolation/ResourceInUse-type error.
# Retry with backoff before giving up.
RETRY_COUNT="${NETWORK_DELETE_RETRIES:-5}"
RETRY_INTERVAL="${NETWORK_DELETE_RETRY_INTERVAL:-15}"

mkdir -p "$(dirname "${RESULT_FILE}")"

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')
echo "[${CSP_NAME}] [${timestamp}] Cleaning up basic resources (VPC: ${VPC_NAME})..."

# ── KeyPair (delete first: no dependency on SG/VPC) ──────────────────────────
kp_err=""
for ((attempt = 1; attempt <= RETRY_COUNT; attempt++)); do
    kp_del=$(curl -u "${SPIDER_AUTH}" -sX DELETE \
      "${SPIDER_URL}/spider/keypair/${KEYPAIR_NAME}" \
      -H 'Content-Type: application/json' \
      -d "{\"ConnectionName\": \"${CONNECTION_NAME}\"}" 2>&1)
    kp_err=$(echo "${kp_del}" | jq -r '.message // empty' 2>/dev/null)

    [[ -z "${kp_err}" ]] && break

    echo "[${CSP_NAME}] KeyPair '${KEYPAIR_NAME}' delete attempt ${attempt}/${RETRY_COUNT} failed: ${kp_err}"
    [[ ${attempt} -lt ${RETRY_COUNT} ]] && sleep "${RETRY_INTERVAL}"
done

if [[ -n "${kp_err}" ]]; then
    echo "[${CSP_NAME}] KeyPair '${KEYPAIR_NAME}' delete: ${kp_err}"
    kp_status="NOT_FOUND_OR_ERROR"
else
    echo "[${CSP_NAME}] KeyPair '${KEYPAIR_NAME}' deleted."
    kp_status="DELETED"
fi

# Remove the saved PrivateKey file (see common-network-prepare.sh), if any.
KEY_DIR="${KEY_DIR:-/tmp/vm_publicip_keys}"
KEY_FILE="${KEY_DIR}/${CONNECTION_NAME}.pem"
rm -f "${KEY_FILE}"

# ── Security Group (delete before VPC: VPC delete would otherwise fail while it's attached) ──
sg_err=""
for ((attempt = 1; attempt <= RETRY_COUNT; attempt++)); do
    sg_del=$(curl -u "${SPIDER_AUTH}" -sX DELETE \
      "${SPIDER_URL}/spider/securitygroup/${SG_NAME}" \
      -H 'Content-Type: application/json' \
      -d "{\"ConnectionName\": \"${CONNECTION_NAME}\"}" 2>&1)
    sg_err=$(echo "${sg_del}" | jq -r '.message // empty' 2>/dev/null)

    [[ -z "${sg_err}" ]] && break

    echo "[${CSP_NAME}] SecurityGroup '${SG_NAME}' delete attempt ${attempt}/${RETRY_COUNT} failed: ${sg_err}"
    [[ ${attempt} -lt ${RETRY_COUNT} ]] && sleep "${RETRY_INTERVAL}"
done

if [[ -n "${sg_err}" ]]; then
    echo "[${CSP_NAME}] SecurityGroup '${SG_NAME}' delete: ${sg_err}"
    sg_status="NOT_FOUND_OR_ERROR"
else
    echo "[${CSP_NAME}] SecurityGroup '${SG_NAME}' deleted."
    sg_status="DELETED"
fi

# ── VPC/Subnet ────────────────────────────────────────────────────────────────
vpc_err=""
for ((attempt = 1; attempt <= RETRY_COUNT; attempt++)); do
    vpc_del=$(curl -u "${SPIDER_AUTH}" -sX DELETE \
      "${SPIDER_URL}/spider/vpc/${VPC_NAME}" \
      -H 'Content-Type: application/json' \
      -d "{\"ConnectionName\": \"${CONNECTION_NAME}\"}" 2>&1)
    vpc_err=$(echo "${vpc_del}" | jq -r '.message // empty' 2>/dev/null)

    [[ -z "${vpc_err}" ]] && break

    echo "[${CSP_NAME}] VPC '${VPC_NAME}' delete attempt ${attempt}/${RETRY_COUNT} failed: ${vpc_err}"
    [[ ${attempt} -lt ${RETRY_COUNT} ]] && sleep "${RETRY_INTERVAL}"
done

if [[ -n "${vpc_err}" ]]; then
    echo "[${CSP_NAME}] VPC '${VPC_NAME}' delete: ${vpc_err}"
    vpc_status="NOT_FOUND_OR_ERROR"
else
    echo "[${CSP_NAME}] VPC '${VPC_NAME}' deleted."
    vpc_status="DELETED"
fi

end_time=$(date +%s)
elapsed=$((end_time - start_time))

echo "[${CSP_NAME}] Done. VPC: ${vpc_status}, SG: ${sg_status}, KeyPair: ${kp_status}"
echo "${CSP_NAME}|${vpc_status}|${sg_status}|${kp_status}|${elapsed}s" > "${RESULT_FILE}"
