#!/bin/bash

# IBM Cloud RDBMS Test Script
# Note: Subnet not required
# Author: CB-Spider Team

export CSP_NAME="IBM"
export CONNECTION_NAME="ibm-us-east-1-config"
export RDBMS_NAME="cb-spider-mysql-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_ibm.txt"

export CREATE_JSON='{
  "ConnectionName": "ibm-us-east-1-config",
  "ReqInfo": {
    "Name": "cb-spider-mysql-test",
    "VPCName": "vpc-01",
    "DBEngine": "mysql",
    "DBEngineVersion": "8.4",
    "DBInstanceSpec": "multitenant",
    "StorageType": "standard",
    "StorageSize": "30",
    "MasterUserName": "admin",
    "MasterUserPassword": "Passwordspider123",
    "PublicAccess": true,
    "TagList": [
      {"Key": "env",        "Value": "test"},
      {"Key": "managed-by", "Value": "cb-spider"}
    ]
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
