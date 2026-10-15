#!/bin/bash

# Alibaba VM AssignPublicIP=false Test Script
# Alibaba rotates its public Ubuntu image monthly (e.g. alibase_20260413.vhd ->
# alibase_20260501.vhd), so instead of a fixed ImageName this script resolves
# the lexicographically latest public image whose name starts with
# IMAGE_NAME_PREFIX via GET /spider/vmimage (same approach as
# ../alibaba-vm-publicip-test.sh).
# Author: CB-Spider Team

export CSP_NAME="ALIBABA"
export CONNECTION_NAME="alibaba-beijing-config"
export VM_NAME="cb-spider-nopublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_nopublicip_results}/result_alibaba.txt"

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
IMAGE_NAME_PREFIX="ubuntu_24_04_x64_20G_alibase_"

mkdir -p "$(dirname "${RESULT_FILE}")"

echo "[${CSP_NAME}] Resolving latest public image with prefix '${IMAGE_NAME_PREFIX}'..."
image_list=$(curl -u "${SPIDER_AUTH}" -s \
  "${SPIDER_URL}/spider/vmimage?ConnectionName=${CONNECTION_NAME}" 2>&1)

resolved_image=$(echo "${image_list}" | jq -r --arg p "${IMAGE_NAME_PREFIX}" '
  .image[]? | (.Name // .IId.NameId // "") | select(startswith($p))
' 2>/dev/null | sort | tail -n1)

if [[ -z "${resolved_image}" ]]; then
    echo "[${CSP_NAME}] ERROR: no public image found with prefix '${IMAGE_NAME_PREFIX}'"
    echo "${CSP_NAME}|FAIL|IMAGE_RESOLVE_ERROR|-|-|-|no public image found with prefix '${IMAGE_NAME_PREFIX}'" > "${RESULT_FILE}"
    exit 1
fi
echo "[${CSP_NAME}] Resolved image: ${resolved_image}"

export CREATE_JSON="{
  \"ConnectionName\": \"alibaba-beijing-config\",
  \"ReqInfo\": {
    \"Name\": \"cb-spider-nopublicip-test\",
    \"ImageName\": \"${resolved_image}\",
    \"VPCName\": \"vpc-01\",
    \"SubnetName\": \"subnet-01\",
    \"SecurityGroupNames\": [\"sg-01\"],
    \"VMSpecName\": \"ecs.c9i.large\",
    \"KeyPairName\": \"keypair-01\",
    \"AssignPublicIP\": false
  }
}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-false-test.sh"
