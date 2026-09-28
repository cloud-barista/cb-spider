#!/bin/bash

# CB-Spider RDBMS TLS Connection Common Test Script
#
# Thin wrapper around ./tlsprobe (a standalone Go program) which does all the actual work:
# fetching the instance's endpoint/CA cert from CB-Spider's REST API and running the
# 4-scenario TLS matrix directly via go-sql-driver/mysql. See tlsprobe/main.go for why this
# replaced an earlier version that shelled out to the `mysql` CLI (unreliable across
# machines/OSes) plus curl/jq.
#
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME              - Display name (e.g., AWS)
#   CONNECTION_NAME       - Spider connection config name
#   RDBMS_NAME            - RDBMS instance name
#   MASTER_USER_PASSWORD  - MasterUserPassword used when creating the RDBMS instance
#   RESULT_FILE           - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL          - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH         - Basic auth credentials (default: admin:****)
#   CONNECT_TIMEOUT     - Per-attempt connection timeout in seconds (default: 10)
#   INTER_ATTEMPT_DELAY - Delay between scenario attempts in seconds (default: 3) --
#                         some networks throttle back-to-back new connections with no gap
#                         at all; see tlsprobe/main.go for what this works around.
#
# Result file format (8 fields): CSP|RequireSecureTransport|TLSAvailable|S1|S2|S3|S4|Elapsed
# Per-scenario values: PASS | FAIL | SKIP | N/A -- see ../README.md
# Scenarios are ordered weakest -> strongest security:
#   S1: tls=false, no CA (no encryption)
#   S2: tls=skip-verify (encrypted, no verification at all)
#   S3: VERIFY_CA equivalent (chain trust only, no hostname check)
#   S4: tls=true, CA registered (chain + hostname verification -- strongest)
# S1 always runs, even when TLSAvailable=OFF (it's the only mode a TLS-less instance
# supports); S2-S4 are N/A in that case.

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
CONNECT_TIMEOUT="${CONNECT_TIMEOUT:-10}"
INTER_ATTEMPT_DELAY="${INTER_ATTEMPT_DELAY:-3}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$(dirname "${RESULT_FILE}")"

if ! command -v go > /dev/null 2>&1; then
    echo "[${CSP_NAME}] ERROR: 'go' toolchain not found in PATH -- required to run tlsprobe (see ${SCRIPT_DIR}/tlsprobe)."
    echo "${CSP_NAME}|N/A|N/A|SKIP|SKIP|SKIP|SKIP|0s" > "${RESULT_FILE}"
    exit 1
fi

exec go run "${SCRIPT_DIR}/tlsprobe" \
    -spider-url "${SPIDER_URL}" \
    -spider-auth "${SPIDER_AUTH}" \
    -connection "${CONNECTION_NAME}" \
    -rdbms "${RDBMS_NAME}" \
    -password "${MASTER_USER_PASSWORD}" \
    -csp-name "${CSP_NAME}" \
    -timeout "${CONNECT_TIMEOUT}s" \
    -inter-attempt-delay "${INTER_ATTEMPT_DELAY}s" \
    -result-file "${RESULT_FILE}"
