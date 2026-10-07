#!/bin/bash

# CB-Spider RDBMS Charset Configuration Common Test Script
#
# Thin wrapper around ./charsetprobe (a standalone Go program) which does all the actual work:
# creating a database via CB-Spider's Charset parameter, then connecting directly via
# go-sql-driver/mysql to verify the charset actually took effect -- both via
# information_schema.SCHEMATA metadata and a functional INSERT/SELECT round-trip of a string that
# only utf8mb4 can hold. See charsetprobe/main.go for the full rationale and ../tls-test/tlsprobe
# for why a Go binary replaces shelling out to the `mysql` CLI/curl/jq.
#
# Scope: all 9 RDBMS-capable CSPs, via three different mechanisms (see RDBMSManager.go):
#   - Azure, GCP, OpenStack, Alibaba: rdbmsDatabaseOptionsManager (native API)
#   - Tencent: rdbmsDatabaseCharsetManager (native API for charset; a Collation -- see
#     ../collation-test -- is applied via a follow-up SQL ALTER DATABASE)
#   - NCP: rdbmsDatabaseSQLStatementBuilder (NCP's `sys.ncp_create_db` stored procedure over SQL)
#   - NHN: rdbmsDatabaseSQLFallbackEligible (plain SQL CREATE DATABASE -- requires the instance
#     owner to have separately enabled "Direct Control" in the NHN console; see
#     nhn-charset-test.sh)
#   - AWS, IBM: plain SQL fallback (no CSP-native database API at all)
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
#   CSP|CreateUtf8mb4|MetaUtf8mb4|FuncUtf8mb4|CreateLatin1|MetaLatin1|FuncLatin1|Elapsed
# Per-check values: PASS | FAIL | SKIP -- see README.md
#   CreateX  - POST .../databases with Charset=X returned "created"
#   MetaX    - information_schema.SCHEMATA.DEFAULT_CHARACTER_SET_NAME for the new DB equals X
#   FuncX    - functional round-trip behaves as expected for X (exact match for utf8mb4;
#              rejected-or-mangled, never an exact match, for latin1)

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
TIMEOUT="${TIMEOUT:-15}"
API_TIMEOUT="${API_TIMEOUT:-120}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$(dirname "${RESULT_FILE}")"

if ! command -v go > /dev/null 2>&1; then
    echo "[${CSP_NAME}] ERROR: 'go' toolchain not found in PATH -- required to run charsetprobe (see ${SCRIPT_DIR}/charsetprobe)."
    echo "${CSP_NAME}|SKIP|SKIP|SKIP|SKIP|SKIP|SKIP|0s" > "${RESULT_FILE}"
    exit 1
fi

exec go run "${SCRIPT_DIR}/charsetprobe" \
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
