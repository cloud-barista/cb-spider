#!/bin/bash

# NHN Cloud RDBMS Test Script (MariaDB)
# Note: Subnet required. Uses NHN RDS for MariaDB endpoint.
# Author: CB-Spider Team

export CSP_NAME="NHN"
export CONNECTION_NAME="nhn-korea-pangyo1-config"
export RDBMS_NAME="cb-spider-mariadb-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_nhn.txt"

export CREATE_JSON='{
  "ConnectionName": "nhn-korea-pangyo1-config",
  "ReqInfo": {
    "Name": "cb-spider-mariadb-test",
    "VPCName": "vpc-01",
    "SubnetNames": ["subnet-01"],
    "DBEngine": "mariadb",
    "DBEngineVersion": "MARIADB_V101118",
    "DBSpec": "m2.c2m4",
    "StorageSize": "20",
    "MasterUserName": "myadmin",
    "MasterUserPassword": "Password123!",
    "PublicAccess": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
