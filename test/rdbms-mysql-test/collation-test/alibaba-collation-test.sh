#!/bin/bash

# Alibaba RDBMS Collation Configuration Test Script
export CSP_NAME="ALIBABA"
export CONNECTION_NAME="alibaba-beijing-config"
export RDBMS_NAME="cb-spider-mysql-test"
export MASTER_USER_NAME="myadmin"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_collation_results}/result_alibaba.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-collation-test.sh"
