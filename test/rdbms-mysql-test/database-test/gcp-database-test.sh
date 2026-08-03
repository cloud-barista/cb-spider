#!/bin/bash

# GCP RDBMS Database Management Test Script
export CSP_NAME="GCP"
export CONNECTION_NAME="gcp-iowa-config"
export RDBMS_NAME="cb-spider-mysql-test"
export MASTER_USER_PASSWORD="Password123!"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_mgmt_results}/result_gcp.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-database-test.sh"
