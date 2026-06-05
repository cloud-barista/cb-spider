#!/bin/bash

# OpenStack RDBMS Test Script
# Note: Subnet not required
# Author: CB-Spider Team

export CSP_NAME="OPENSTACK"
export CONNECTION_NAME="openstack-config01"
export RDBMS_NAME="cb-spider-mysql-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_openstack.txt"

export CREATE_JSON='{
  "ConnectionName": "openstack-config01",
  "ReqInfo": {
    "Name": "cb-spider-mysql-test",
    "VPCName": "vpc-01",
    "DBEngine": "mysql",
    "DBEngineVersion": "5.7.29",
    "DBInstanceSpec": "m1.small",
    "StorageSize": "20",
    "MasterUserName": "myadmin",
    "MasterUserPassword": "Password123!",
    "PublicAccess": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
