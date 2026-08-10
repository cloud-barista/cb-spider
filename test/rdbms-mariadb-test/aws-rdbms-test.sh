#!/bin/bash

# AWS RDBMS Test Script (MariaDB)
# Note: Requires 2+ subnets in different Availability Zones (SubnetGroup requirement)
# Author: CB-Spider Team

export CSP_NAME="AWS"
export CONNECTION_NAME="aws-config01"
export RDBMS_NAME="cb-spider-mariadb-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_aws.txt"

export CREATE_JSON='{
  "ConnectionName": "aws-config01",
  "ReqInfo": {
    "Name": "cb-spider-mariadb-test",
    "VPCName": "vpc-01",
    "DBEngine": "mariadb",
    "DBEngineVersion": "10.6",
    "DBInstanceSpec": "db.t3.medium",
    "StorageSize": "100",
    "SubnetNames": ["subnet-01", "subnet-02"],
    "SecurityGroupNames": ["sg-01"],
    "MasterUserName": "myadmin",
    "MasterUserPassword": "Password123!",
    "PublicAccess": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
