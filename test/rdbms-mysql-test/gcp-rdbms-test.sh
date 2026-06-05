#!/bin/bash

# GCP RDBMS Test Script
# Note: Subnet not required. Creation may take several minutes.
# Author: CB-Spider Team

export CSP_NAME="GCP"
export CONNECTION_NAME="gcp-iowa-config"
export RDBMS_NAME="cb-spider-mysql-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_gcp.txt"

export CREATE_JSON='{
  "ConnectionName": "gcp-iowa-config",
  "ReqInfo": {
    "Name": "cb-spider-mysql-test",
    "VPCName": "vpc-01",
    "DBEngine": "mysql",
    "DBEngineVersion": "8.0",
    "DBInstanceSpec": "db-custom-2-8192",
    "StorageSize": "20",
    "MasterUserName": "myadmin",
    "MasterUserPassword": "Password123!",
    "PublicAccess": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
