#!/bin/bash

# Alibaba Cloud RDBMS Test Script (MariaDB)
# Note: Subnet required
# Author: CB-Spider Team

export CSP_NAME="ALIBABA"
export CONNECTION_NAME="alibaba-beijing-config"
export RDBMS_NAME="cb-spider-mariadb-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_alibaba.txt"

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"

# Fetch DBSpecOptions from MetaInfo to pick a spec valid for the default
# storage type (cloud_essd). Hardcoded specs (e.g. rds.mariadb.s4.large) are only
# valid for local_ssd and will fail with InvalidDBInstanceClass.NotFound on cloud_essd.
meta_resp=$(curl -u "${SPIDER_AUTH}" -sX GET \
    "${SPIDER_URL}/spider/rdbmsmetainfo?DBEngine=mariadb&ConnectionName=${CONNECTION_NAME}" 2>&1)
db_instance_spec=$(echo "${meta_resp}" | jq -r '.DBSpecOptions[0]? // empty' 2>/dev/null)
if [[ -z "${db_instance_spec}" ]]; then
    db_instance_spec="rds.mariadb.s4.large"
    echo "[${CSP_NAME}] No DBSpecOptions in metainfo; falling back to ${db_instance_spec}"
else
    echo "[${CSP_NAME}] Using DBSpec from metainfo: ${db_instance_spec}"
fi

export CREATE_JSON="{
  \"ConnectionName\": \"${CONNECTION_NAME}\",
  \"ReqInfo\": {
    \"Name\": \"cb-spider-mariadb-test\",
    \"VPCName\": \"vpc-01\",
    \"SubnetNames\": [\"subnet-01\"],
    \"DBEngine\": \"mariadb\",
    \"DBEngineVersion\": \"10.6\",
    \"DBSpec\": \"${db_instance_spec}\",
    \"StorageSize\": \"20\",
    \"MasterUserName\": \"myadmin\",
    \"MasterUserPassword\": \"Password123!\",
    \"PublicAccess\": true
  }
}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
