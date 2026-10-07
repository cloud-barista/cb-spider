#!/bin/bash

# AWS RDBMS Charset Configuration Test Script
export CSP_NAME="AWS"
export CONNECTION_NAME="aws-config01"
export RDBMS_NAME="cb-spider-mysql-test"
export MASTER_USER_NAME="myadmin"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_charset_results}/result_aws.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-charset-test.sh"
