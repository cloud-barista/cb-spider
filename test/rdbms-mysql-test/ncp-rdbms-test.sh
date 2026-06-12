#!/bin/bash

# NCP (Naver Cloud Platform) RDBMS Test Script
# Note: Subnet required. StorageSize and StorageType are not configurable (SupportsStorageSizeConfiguration=false, SupportsStorageTypeSelection=false).
#       G3 (KVM) server generation only. Public domain must be requested via console after creation.
# Author: CB-Spider Team

export CSP_NAME="NCP"
export CONNECTION_NAME="ncp-korea1-config"
export RDBMS_NAME="cb-spider-mysql-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_ncp.txt"

export CREATE_JSON='{
  "ConnectionName": "ncp-korea1-config",
  "ReqInfo": {
    "Name": "cb-spider-mysql-test",
    "VPCName": "vpc-01",
    "SubnetNames": ["subnet-01"],
    "DBInstanceSpec": "SVR.VDBAS.AMD.STAND.C002.M008.NET.SSD.B050.G003",
    "DBEngine": "mysql",
    "DBEngineVersion": "8.0.36",
    "MasterUserName": "myadmin",
    "MasterUserPassword": "Password123!",
    "PublicAccess": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
