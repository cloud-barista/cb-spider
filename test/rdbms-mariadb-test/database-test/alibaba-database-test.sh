#!/bin/bash

# ALIBABA RDBMS Database Management Test Script (MariaDB)
export CSP_NAME="ALIBABA"
export CONNECTION_NAME="alibaba-beijing-config"
export RDBMS_NAME="cb-spider-mariadb-test"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_mgmt_results}/result_alibaba.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-database-test.sh"
