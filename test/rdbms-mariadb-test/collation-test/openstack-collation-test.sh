#!/bin/bash

# OpenStack RDBMS Collation Configuration Test Script (MariaDB)
export CSP_NAME="OPENSTACK"
export CONNECTION_NAME="openstack-config01"
export RDBMS_NAME="cb-spider-mariadb-test"
export MASTER_USER_NAME="myadmin"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_collation_results}/result_openstack.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-collation-test.sh"
