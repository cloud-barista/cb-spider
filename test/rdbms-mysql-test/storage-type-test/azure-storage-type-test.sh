#!/bin/bash

# Azure RDBMS StorageType Test Script
# SKIP: Azure MySQL Flexible Server does not support user-selectable StorageType.
#       storageSku is read-only and set automatically by Azure (always Premium_LRS).
#       SupportsStorageTypeSelection=false in GetMetaInfo.

CSP_NAME="AZURE"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export RESULT_DIR="${RESULT_DIR:-/tmp/st_results_$$}"
mkdir -p "${RESULT_DIR}"

SKIP_REASON="SupportsStorageTypeSelection=false: storageSku is read-only, set automatically by Azure"

echo "[${CSP_NAME}] SKIP - ${SKIP_REASON}"

# Format: CSP|StorageType_Requested|StorageType_Returned|PASS_FAIL|DB_Status|Elapsed|Reason
echo "${CSP_NAME}|N/A|N/A|SKIP|NOT_APPLICABLE|-|${SKIP_REASON}" \
    > "${RESULT_DIR}/result_azure_skip.txt"
