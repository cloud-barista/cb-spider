#!/bin/bash

# CB-Spider RDBMS Database Management Common Test Script
# Flow: CreateDatabase -> ListDatabases -> DeleteDatabase -> ListDatabases(verify)
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
#   SPIDER_URL      - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)
#   DB_NAME         - Test database name to create (default: spidertestdb)
#
# Result file format (7 fields):
#   CSP|CreateDB|ListDB|FoundInList|DeleteDB|VerifyDeleted|Elapsed

format_elapsed() {
    local sec=$1
    if [[ ${sec} -lt 60 ]]; then
        echo "${sec}s"
    else
        echo "$((sec / 60))m$((sec % 60))s"
    fi
}

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
DB_NAME="${DB_NAME:-spidertestdb}"

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

mkdir -p "$(dirname "${RESULT_FILE}")"

r_create="FAIL"
r_list="FAIL"
r_found="NOT_FOUND"
r_delete="FAIL"
r_verify="FAIL"

abort() {
    local label="$1"
    local msg="$2"
    echo "[${CSP_NAME}] ERROR on ${label}: ${msg}"
    elapsed_fmt=$(format_elapsed $(($(date +%s) - start_time)))
    echo "${CSP_NAME}|${r_create}|${r_list}|${r_found}|${r_delete}|${r_verify}|${elapsed_fmt}" \
        > "${RESULT_FILE}"
    exit 1
}

echo "[${CSP_NAME}] [${timestamp}] Starting database management test (RDBMS='${RDBMS_NAME}', DB='${DB_NAME}')..."

# ── Pre-cleanup: delete leftover DB from a previous run (ignore errors) ───────
curl -u "${SPIDER_AUTH}" -sX DELETE \
  "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}/databases/${DB_NAME}" \
  -H 'Content-Type: application/json' \
  -d "{\"ConnectionName\": \"${CONNECTION_NAME}\", \"MasterUserPassword\": \"${MASTER_USER_PASSWORD}\"}" \
  > /dev/null 2>&1

# ── CreateDatabase ────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] CreateDatabase: '${DB_NAME}'"

create_resp=$(curl -u "${SPIDER_AUTH}" -sX POST \
  "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}/databases" \
  -H 'Content-Type: application/json' \
  -d "{
    \"ConnectionName\": \"${CONNECTION_NAME}\",
    \"DatabaseName\": \"${DB_NAME}\",
    \"MasterUserPassword\": \"${MASTER_USER_PASSWORD}\"
  }" 2>&1)

create_msg=$(echo "${create_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ "${create_msg}" == "created" ]]; then
    r_create="PASS"
else
    abort "CreateDatabase" "${create_msg:-unexpected response}"
fi
echo "[${CSP_NAME}] CreateDatabase: ${r_create}"

# ── ListDatabases ─────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] ListDatabases: verifying '${DB_NAME}' is present"

list_resp=$(curl -u "${SPIDER_AUTH}" -sX GET \
  "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}/databases" \
  -H 'Content-Type: application/json' \
  -d "{
    \"ConnectionName\": \"${CONNECTION_NAME}\",
    \"MasterUserPassword\": \"${MASTER_USER_PASSWORD}\"
  }" 2>&1)

list_err=$(echo "${list_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${list_err}" ]]; then
    abort "ListDatabases" "${list_err}"
fi

db_count=$(echo "${list_resp}" | jq -r '.Databases | length // 0' 2>/dev/null)
db_count="${db_count:-0}"
r_list="PASS"

if echo "${list_resp}" | jq -e --arg name "${DB_NAME}" '.Databases[]? | select(. == $name)' > /dev/null 2>&1; then
    r_found="FOUND"
fi
echo "[${CSP_NAME}] ListDatabases: ${r_list} (${db_count} DB(s)), FoundInList: ${r_found}"

# ── DeleteDatabase ────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] DeleteDatabase: '${DB_NAME}'"

delete_resp=$(curl -u "${SPIDER_AUTH}" -sX DELETE \
  "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}/databases/${DB_NAME}" \
  -H 'Content-Type: application/json' \
  -d "{
    \"ConnectionName\": \"${CONNECTION_NAME}\",
    \"MasterUserPassword\": \"${MASTER_USER_PASSWORD}\"
  }" 2>&1)

delete_msg=$(echo "${delete_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ "${delete_msg}" == "deleted" ]]; then
    r_delete="PASS"
else
    abort "DeleteDatabase" "${delete_msg:-unexpected response}"
fi
echo "[${CSP_NAME}] DeleteDatabase: ${r_delete}"

# ── ListDatabases (verify deleted) ───────────────────────────────────────────
echo "[${CSP_NAME}] ListDatabases: verifying '${DB_NAME}' is removed"

verify_resp=$(curl -u "${SPIDER_AUTH}" -sX GET \
  "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}/databases" \
  -H 'Content-Type: application/json' \
  -d "{
    \"ConnectionName\": \"${CONNECTION_NAME}\",
    \"MasterUserPassword\": \"${MASTER_USER_PASSWORD}\"
  }" 2>&1)

verify_err=$(echo "${verify_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${verify_err}" ]]; then
    abort "ListDatabases(verify)" "${verify_err}"
fi

if ! echo "${verify_resp}" | jq -e --arg name "${DB_NAME}" '.Databases[]? | select(. == $name)' > /dev/null 2>&1; then
    r_verify="PASS"
fi
remaining=$(echo "${verify_resp}" | jq -r '.Databases | length // 0' 2>/dev/null)
echo "[${CSP_NAME}] VerifyDeleted: ${r_verify} (${remaining} DB(s) remaining)"

# ── Write Result ──────────────────────────────────────────────────────────────
end_time=$(date +%s)
elapsed_fmt=$(format_elapsed $((end_time - start_time)))

overall="PASS"
for r in "${r_create}" "${r_list}" "${r_found}" "${r_delete}" "${r_verify}"; do
    [[ "${r}" != "PASS" && "${r}" != "FOUND" ]] && overall="FAIL" && break
done

echo "[${CSP_NAME}] Database management test ${overall} (elapsed: ${elapsed_fmt})"
echo "[${CSP_NAME}]   CreateDB=${r_create} ListDB=${r_list} FoundInList=${r_found} DeleteDB=${r_delete} VerifyDeleted=${r_verify}"

# Format: CSP|CreateDB|ListDB|FoundInList|DeleteDB|VerifyDeleted|Elapsed
echo "${CSP_NAME}|${r_create}|${r_list}|${r_found}|${r_delete}|${r_verify}|${elapsed_fmt}" \
  > "${RESULT_FILE}"
