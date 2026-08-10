#!/bin/bash

# AWS RDBMS Tag Management Test Script (MariaDB, SupportsTag=true)
export CSP_NAME="AWS"
export CONNECTION_NAME="aws-config01"
export RDBMS_NAME="cb-spider-mariadb-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_tag_results}/result_aws.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-tag-test.sh"
