#!/bin/bash

# CB-Spider RDBMS Test - Network Prerequisite Prepare Script
# Creates the VPC/Subnet (and Security Group, if requested) required before
# running the RDBMS creation tests in this directory. Idempotent: skips
# creation and reports EXISTS if the resource is already there.
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   VPC_NAME        - VPC name to create
#   VPC_CIDR        - VPC IPv4 CIDR
#   SUBNET_JSON     - JSON array for SubnetInfoList
#                     (e.g. '[{"Name":"subnet-01","IPv4_CIDR":"10.0.1.0/24"}]')
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   SG_NAME         - Security Group name to create (omit to skip; only AWS needs one)
#   SG_RULES_JSON   - JSON array for SecurityRules
#                     (default: allow inbound TCP 3306 from 0.0.0.0/0)
#   SPIDER_URL      - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

# Note: not using "${SG_RULES_JSON:-...}" here — bash ends a ${VAR:-default}
# expansion at the first unescaped '}', so a default containing literal JSON
# braces gets truncated mid-object.
if [[ -z "${SG_RULES_JSON}" ]]; then
    SG_RULES_JSON='[{"Direction":"inbound","IPProtocol":"TCP","FromPort":"3306","ToPort":"3306"}]'
fi

mkdir -p "$(dirname "${RESULT_FILE}")"

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')
echo "[${CSP_NAME}] [${timestamp}] Preparing network resources (VPC: ${VPC_NAME})..."

# ── VPC/Subnet ────────────────────────────────────────────────────────────────
vpc_check=$(curl -u "${SPIDER_AUTH}" -s \
  "${SPIDER_URL}/spider/vpc/${VPC_NAME}?ConnectionName=${CONNECTION_NAME}" 2>&1)
vpc_err=$(echo "${vpc_check}" | jq -r '.message // empty' 2>/dev/null)

if [[ -z "${vpc_err}" ]]; then
    echo "[${CSP_NAME}] VPC '${VPC_NAME}' already exists. Skipping creation."
    vpc_status="EXISTS"
else
    create_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/vpc" \
      -H 'Content-Type: application/json' \
      -d "{
        \"ConnectionName\": \"${CONNECTION_NAME}\",
        \"ReqInfo\": {
          \"Name\": \"${VPC_NAME}\",
          \"IPv4_CIDR\": \"${VPC_CIDR}\",
          \"SubnetInfoList\": ${SUBNET_JSON}
        }
      }" 2>&1)

    create_err=$(echo "${create_resp}" | jq -r '.message // empty' 2>/dev/null)
    if [[ -n "${create_err}" ]]; then
        echo "[${CSP_NAME}] ERROR creating VPC: ${create_err}"
        echo "${CSP_NAME}|VPC_ERROR|SKIPPED|-" > "${RESULT_FILE}"
        exit 1
    fi
    echo "[${CSP_NAME}] VPC '${VPC_NAME}' created."
    vpc_status="CREATED"
fi

# ── Security Group (optional) ────────────────────────────────────────────────
sg_status="SKIPPED"
if [[ -n "${SG_NAME}" ]]; then
    sg_check=$(curl -u "${SPIDER_AUTH}" -s \
      "${SPIDER_URL}/spider/securitygroup/${SG_NAME}?ConnectionName=${CONNECTION_NAME}" 2>&1)
    sg_err=$(echo "${sg_check}" | jq -r '.message // empty' 2>/dev/null)

    if [[ -z "${sg_err}" ]]; then
        echo "[${CSP_NAME}] SecurityGroup '${SG_NAME}' already exists. Skipping creation."
        sg_status="EXISTS"
    else
        sg_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/securitygroup" \
          -H 'Content-Type: application/json' \
          -d "{
            \"ConnectionName\": \"${CONNECTION_NAME}\",
            \"ReqInfo\": {
              \"Name\": \"${SG_NAME}\",
              \"VPCName\": \"${VPC_NAME}\",
              \"SecurityRules\": ${SG_RULES_JSON}
            }
          }" 2>&1)

        sg_create_err=$(echo "${sg_resp}" | jq -r '.message // empty' 2>/dev/null)
        if [[ -n "${sg_create_err}" ]]; then
            echo "[${CSP_NAME}] ERROR creating SecurityGroup: ${sg_create_err}"
            echo "${CSP_NAME}|${vpc_status}|SG_ERROR|-" > "${RESULT_FILE}"
            exit 1
        fi
        echo "[${CSP_NAME}] SecurityGroup '${SG_NAME}' created."
        sg_status="CREATED"
    fi
fi

end_time=$(date +%s)
elapsed=$((end_time - start_time))

echo "[${CSP_NAME}] Done. VPC: ${vpc_status}, SG: ${sg_status}"
echo "${CSP_NAME}|${vpc_status}|${sg_status}|${elapsed}s" > "${RESULT_FILE}"
