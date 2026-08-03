#!/bin/bash

# IBM RDBMS StorageType Test Script
# SKIP: IBM Cloud Databases does not support user-selectable StorageType.
#       Its Global Catalog plans ("standard" / "standard-gen2") select the
#       Gen1 vs Gen2 platform generation, not a storage type - IBM Cloud
#       Databases only lets you configure disk SIZE and host_flavor
#       (DBInstanceSpec), never the underlying storage media/type.
#       CB-Spider only provisions Gen1 ("standard").
#       SupportsStorageTypeSelection=false in GetMetaInfo.

CSP_NAME="IBM"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export RESULT_DIR="${RESULT_DIR:-/tmp/st_results_$$}"
mkdir -p "${RESULT_DIR}"

SKIP_REASON="SupportsStorageTypeSelection=false: IBM Cloud Databases has no storage type selection"

echo "[${CSP_NAME}] SKIP - ${SKIP_REASON}"

# Format: CSP|StorageType_Requested|StorageType_Returned|PASS_FAIL|DB_Status|Elapsed|Reason
echo "${CSP_NAME}|N/A|N/A|SKIP|NOT_APPLICABLE|-|${SKIP_REASON}" \
    > "${RESULT_DIR}/result_ibm_skip.txt"
