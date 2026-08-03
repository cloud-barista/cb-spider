#!/bin/bash

# Alibaba RDBMS Tag Management Test Script (SupportsTag=true)
export CSP_NAME="ALIBABA"
export CONNECTION_NAME="alibaba-beijing-config"
export RDBMS_NAME="cb-spider-mysql-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_tag_results}/result_alibaba.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-rdbms-tag-test.sh"
