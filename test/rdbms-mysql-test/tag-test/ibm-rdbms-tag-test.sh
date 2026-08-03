#!/bin/bash

# IBM RDBMS Tag Management Test Script (SupportsTag=true)
export CSP_NAME="IBM"
export CONNECTION_NAME="ibm-us-east-1-config"
export RDBMS_NAME="cb-spider-mysql-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_tag_results}/result_ibm.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-tag-test.sh"
