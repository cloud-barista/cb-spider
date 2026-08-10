#!/bin/bash

# CB-Spider RDBMS Test - Network Prerequisite Cleanup Script
# Deletes the Security Group (if any) and then the VPC/Subnet created by
# common-network-prepare.sh. Safe to run even if the resources don't exist.
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   VPC_NAME        - VPC name to delete
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   SG_NAME         - Security Group name to delete (omit if none was created)
#   SPIDER_URL      - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

# CSPs release the network resources (ENI/IP/subnet association) that were held
# by the just-deleted RDBMS instance asynchronously, so a delete attempted right
# after RDBMS deletion can hit a transient DependencyViolation/ResourceInUse-type
# error. Some CSPs (e.g. Alibaba: RDS -> vSwitch dependency release) have been
# observed to take several minutes, well past a handful of quick retries, so we
# back off linearly (capped) instead of a fixed short interval.
RETRY_COUNT="${NETWORK_DELETE_RETRIES:-12}"
RETRY_INTERVAL="${NETWORK_DELETE_RETRY_INTERVAL:-15}"
RETRY_MAX_INTERVAL="${NETWORK_DELETE_RETRY_MAX_INTERVAL:-60}"

# retry_delay_sec N -> sleep duration before the (N+1)th attempt: grows linearly
# with the attempt number, capped at RETRY_MAX_INTERVAL, so early retries stay
# fast while later ones give slow CSP-side cleanup enough time to finish.
retry_delay_sec() {
    local attempt="$1"
    local delay=$((RETRY_INTERVAL * attempt))
    if [[ ${delay} -gt ${RETRY_MAX_INTERVAL} ]]; then
        delay="${RETRY_MAX_INTERVAL}"
    fi
    echo "${delay}"
}

mkdir -p "$(dirname "${RESULT_FILE}")"

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')
echo "[${CSP_NAME}] [${timestamp}] Cleaning up network resources (VPC: ${VPC_NAME})..."

# ── Security Group (optional, delete first: VPC delete would otherwise fail while it's attached) ──
sg_status="SKIPPED"
if [[ -n "${SG_NAME}" ]]; then
    sg_err=""
    for ((attempt = 1; attempt <= RETRY_COUNT; attempt++)); do
        sg_del=$(curl -u "${SPIDER_AUTH}" -sX DELETE \
          "${SPIDER_URL}/spider/securitygroup/${SG_NAME}" \
          -H 'Content-Type: application/json' \
          -d "{\"ConnectionName\": \"${CONNECTION_NAME}\"}" 2>&1)
        sg_err=$(echo "${sg_del}" | jq -r '.message // empty' 2>/dev/null)

        [[ -z "${sg_err}" ]] && break

        if [[ ${attempt} -lt ${RETRY_COUNT} ]]; then
            delay=$(retry_delay_sec "${attempt}")
            echo "[${CSP_NAME}] SecurityGroup '${SG_NAME}' delete attempt ${attempt}/${RETRY_COUNT} failed: ${sg_err} (retrying in ${delay}s)"
            sleep "${delay}"
        else
            echo "[${CSP_NAME}] SecurityGroup '${SG_NAME}' delete attempt ${attempt}/${RETRY_COUNT} failed: ${sg_err}"
        fi
    done

    if [[ -n "${sg_err}" ]]; then
        echo "[${CSP_NAME}] SecurityGroup '${SG_NAME}' delete: ${sg_err}"
        sg_status="NOT_FOUND_OR_ERROR"
    else
        echo "[${CSP_NAME}] SecurityGroup '${SG_NAME}' deleted."
        sg_status="DELETED"
    fi
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

    if [[ ${attempt} -lt ${RETRY_COUNT} ]]; then
        delay=$(retry_delay_sec "${attempt}")
        echo "[${CSP_NAME}] VPC '${VPC_NAME}' delete attempt ${attempt}/${RETRY_COUNT} failed: ${vpc_err} (retrying in ${delay}s)"
        sleep "${delay}"
    else
        echo "[${CSP_NAME}] VPC '${VPC_NAME}' delete attempt ${attempt}/${RETRY_COUNT} failed: ${vpc_err}"
    fi
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

echo "[${CSP_NAME}] Done. VPC: ${vpc_status}, SG: ${sg_status}"
echo "${CSP_NAME}|${vpc_status}|${sg_status}|${elapsed}s" > "${RESULT_FILE}"
