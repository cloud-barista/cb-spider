#!/bin/bash

# Tencent RDBMS Database Management Test Script
export CSP_NAME="TENCENT"
export CONNECTION_NAME="tencent-beijing3-config"
export RDBMS_NAME="cb-spider-mysql-test"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_mgmt_results}/result_tencent.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-database-test.sh"
