#!/bin/bash

# CB-Spider RDBMS Collation Configuration Common Test Script (MariaDB engine)
#
# Thin wrapper around ./collationprobe (a standalone Go program) which does all the actual work:
# creating a database via CB-Spider's Collation parameter (Charset held fixed at utf8mb4 so only
# collation varies), then connecting directly via go-sql-driver/mysql to verify the collation
# actually took effect -- both via information_schema.SCHEMATA metadata and a functional
# case-sensitivity comparison. See collationprobe/main.go for the full rationale and
# ../charset-test/charsetprobe (this suite's sibling) for why a Go binary replaces shelling out to
# the `mysql` CLI/curl/jq.
#
# Scope: only CSPs that (a) support MariaDB at all and (b) have the Collation parameter wired up
# in CB-Spider today: AWS, Alibaba, OpenStack, NHN -- the same scope as ../charset-test, since both
# parameters were wired up together. See ../charset-test/common-charset-test.sh for why
# Azure/GCP/IBM/Tencent/NCP are not included. NHN's SQL path requires the instance owner to have
# separately enabled "Direct Control" in the NHN console -- see nhn-collation-test.sh.
#
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME              - Display name (e.g., AWS)
#   CONNECTION_NAME       - Spider connection config name
#   RDBMS_NAME            - RDBMS instance name
#   MASTER_USER_NAME      - MasterUserName used when creating the RDBMS instance (CB-Spider's
#                           GET /spider/rdbms/{Name} no longer returns it -- see
#                           api-runtime/common-runtime/RDBMSManager.go's redactRDBMSMasterCredentials)
#   MASTER_USER_PASSWORD  - MasterUserPassword used when creating the RDBMS instance
#   RESULT_FILE           - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL      - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)
#   TIMEOUT         - Per-operation timeout in seconds for direct SQL connections
#                     (dial/ping/query/exec) -- these are always fast once the server is
#                     reachable (default: 15)
#   API_TIMEOUT     - Per-call timeout in seconds for CB-Spider REST API calls
#                     (GetRDBMS/CreateDatabase/DeleteDatabase). Deliberately generous: Azure's
#                     CreateDatabase polls an ARM async operation to completion before
#                     responding, observed to take over a minute -- this is normal CSP latency,
#                     not a CB-Spider bug (default: 120)
#
# Result file format (8 fields):
#   CSP|CreateCI|MetaCI|FuncCI|CreateBin|MetaBin|FuncBin|Elapsed
# Per-check values: PASS | FAIL | SKIP -- see README.md
#   CreateX  - POST .../databases with Charset=utf8mb4, Collation=X returned "created"
#   MetaX    - information_schema.SCHEMATA.DEFAULT_COLLATION_NAME for the new DB equals X
#   FuncX    - functional case-comparison behaves as expected for X (case-insensitive match for
#              utf8mb4_general_ci; no match for utf8mb4_bin)

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
TIMEOUT="${TIMEOUT:-15}"
API_TIMEOUT="${API_TIMEOUT:-120}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$(dirname "${RESULT_FILE}")"

if ! command -v go > /dev/null 2>&1; then
    echo "[${CSP_NAME}] ERROR: 'go' toolchain not found in PATH -- required to run collationprobe (see ${SCRIPT_DIR}/collationprobe)."
    echo "${CSP_NAME}|SKIP|SKIP|SKIP|SKIP|SKIP|SKIP|0s" > "${RESULT_FILE}"
    exit 1
fi

exec go run "${SCRIPT_DIR}/collationprobe" \
    -spider-url "${SPIDER_URL}" \
    -spider-auth "${SPIDER_AUTH}" \
    -connection "${CONNECTION_NAME}" \
    -rdbms "${RDBMS_NAME}" \
    -username "${MASTER_USER_NAME}" \
    -password "${MASTER_USER_PASSWORD}" \
    -csp-name "${CSP_NAME}" \
    -timeout "${TIMEOUT}s" \
    -api-timeout "${API_TIMEOUT}s" \
    -result-file "${RESULT_FILE}"
