#!/bin/bash

# NCP RDBMS StorageType Test Script
# SKIP: NCP MySQL G3 does not support user-selectable StorageType.
#       SSD is applied automatically; DataStorageTypeCode must NOT be specified.
#       SupportsStorageTypeSelection=false in GetMetaInfo.

CSP_NAME="NCP"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export RESULT_DIR="${RESULT_DIR:-/tmp/st_results_$$}"
mkdir -p "${RESULT_DIR}"

SKIP_REASON="SupportsStorageTypeSelection=false: NCP G3 applies SSD automatically, StorageType cannot be specified"

echo "[${CSP_NAME}] SKIP - ${SKIP_REASON}"

# Format: CSP|StorageType_Requested|StorageType_Returned|PASS_FAIL|DB_Status|Elapsed|Reason
echo "${CSP_NAME}|N/A|N/A|SKIP|NOT_APPLICABLE|-|${SKIP_REASON}" \
    > "${RESULT_DIR}/result_ncp_skip.txt"
