#!/bin/bash

# Azure RDBMS Test Script
# Note: Subnet not required for Azure Flexible Server
# Author: CB-Spider Team

export CSP_NAME="AZURE"
export CONNECTION_NAME="azure-koreacentral-config"
export RDBMS_NAME="cb-spider-mysql-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_azure.txt"

export CREATE_JSON='{
  "ConnectionName": "azure-koreacentral-config",
  "ReqInfo": {
    "Name": "cb-spider-mysql-test",
    "VPCName": "vpc-01",
    "DBEngine": "mysql",
    "DBEngineVersion": "8.0.21",
    "DBInstanceSpec": "Standard_B1ms",
    "StorageSize": "20",
    "MasterUserName": "myadmin",
    "MasterUserPassword": "Password123!",
    "PublicAccess": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
