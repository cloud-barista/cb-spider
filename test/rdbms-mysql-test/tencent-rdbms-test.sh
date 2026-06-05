#!/bin/bash

# Tencent Cloud RDBMS Test Script
# Note: Subnet required. DBInstanceSpec is memory size in MB (e.g., 8000 = 8GB).
# Author: CB-Spider Team

export CSP_NAME="TENCENT"
export CONNECTION_NAME="tencent-beijing3-config"
export RDBMS_NAME="cb-spider-mysql-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_tencent.txt"

export CREATE_JSON='{
  "ConnectionName": "tencent-beijing3-config",
  "ReqInfo": {
    "Name": "cb-spider-mysql-test",
    "VPCName": "vpc-01",
    "SubnetNames": ["subnet-01"],
    "DBEngine": "mysql",
    "DBEngineVersion": "8.0",
    "DBInstanceSpec": "8000",
    "StorageSize": "50",
    "StorageType": "CLOUD_HSSD",
    "MasterUserName": "root",
    "MasterUserPassword": "Password123!",
    "PublicAccess": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
