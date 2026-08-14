#!/bin/bash

# OpenStack RDBMS Test Script (MariaDB)
# Note: MariaDB support depends on OpenStack Trove datastore configuration.
#       Verify available datastores with: trove datastore-list
# Author: CB-Spider Team

export CSP_NAME="OPENSTACK"
export CONNECTION_NAME="openstack-config01"
export RDBMS_NAME="cb-spider-mariadb-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_results}/result_openstack.txt"

export CREATE_JSON='{
  "ConnectionName": "openstack-config01",
  "ReqInfo": {
    "Name": "cb-spider-mariadb-test",
    "VPCName": "vpc-01",
    "DBEngine": "mariadb",
    "DBEngineVersion": "10.6",
    "DBSpec": "m1.small",
    "StorageSize": "20",
    "MasterUserName": "myadmin",
    "MasterUserPassword": "Password123!",
    "PublicAccess": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-test.sh"
