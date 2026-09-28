#!/bin/bash

# IBM RDBMS TLS Connection Test Script
export CSP_NAME="IBM"
export CONNECTION_NAME="ibm-us-east-1-config"
export RDBMS_NAME="cb-spider-mysql-test"
export MASTER_USER_PASSWORD="Passwordspider123"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_tls_results}/result_ibm.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-tls-test.sh"
