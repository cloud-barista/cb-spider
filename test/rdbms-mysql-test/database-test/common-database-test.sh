#!/bin/bash

# CB-Spider RDBMS Database Management Common Test Script
# Flow: CreateDatabase -> ListDatabases -> DeleteDatabase -> ListDatabases(verify)
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME              - Display name (e.g., AWS)
#   CONNECTION_NAME       - Spider connection config name
#   RDBMS_NAME            - RDBMS instance name
#   MASTER_USER_NAME      - MasterUserName used when creating the RDBMS instance (CB-Spider's
#                           GET /spider/rdbms/{Name} no longer returns it -- see
#                           api-runtime/common-runtime/RDBMSManager.go's redactRDBMSMasterCredentials;
#                           only needed by the SQL fallback path, e.g. AWS/IBM -- ignored by CSPs
#                           with CSP-native database management)
#   MASTER_USER_PASSWORD  - MasterUserPassword used when creating the RDBMS instance
#   RESULT_FILE           - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL      - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH     - Basic auth credentials (default: admin:****)
#   DB_NAME         - Test database name to create (default: spidertestdb)
#   MAX_RETRIES     - Retry attempts per API call on transient failure (default: 3)
#   RETRY_DELAY     - Seconds to wait between retries (default: 3)
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

MAX_RETRIES="${MAX_RETRIES:-3}"
RETRY_DELAY="${RETRY_DELAY:-3}"

# run_with_retry retries a curl-based API call up to MAX_RETRIES times when it fails for
# transient reasons (e.g. a CSP-side proxy briefly returning "Service Unavailable" right after
# a DDL operation -- observed in practice on IBM under parallel multi-CSP load, not tied to any
# particular step). $1 is a label for logging, $2 is the name of a function that performs one
# attempt: it must set the global `attempt_resp` to the raw response body and return 0 if the
# attempt should be treated as successful, 1 if it should be retried.
run_with_retry() {
    local label="$1"
    local attempt_fn="$2"
    local n
    for ((n = 1; n <= MAX_RETRIES; n++)); do
        if "${attempt_fn}"; then
            return 0
        fi
        if [[ ${n} -lt ${MAX_RETRIES} ]]; then
            echo "[${CSP_NAME}] ${label}: attempt ${n}/${MAX_RETRIES} failed, retrying in ${RETRY_DELAY}s..."
            sleep "${RETRY_DELAY}"
        fi
    done
    return 1
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
  -d "{\"ConnectionName\": \"${CONNECTION_NAME}\", \"MasterUserName\": \"${MASTER_USER_NAME}\", \"MasterUserPassword\": \"${MASTER_USER_PASSWORD}\"}" \
  > /dev/null 2>&1

# ── CreateDatabase ────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] CreateDatabase: '${DB_NAME}'"

attempt_create_database() {
    attempt_resp=$(curl -u "${SPIDER_AUTH}" -sX POST \
      "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}/databases" \
      -H 'Content-Type: application/json' \
      -d "{
        \"ConnectionName\": \"${CONNECTION_NAME}\",
        \"DatabaseName\": \"${DB_NAME}\",
        \"MasterUserName\": \"${MASTER_USER_NAME}\",
        \"MasterUserPassword\": \"${MASTER_USER_PASSWORD}\"
      }" 2>&1)
    [[ "$(echo "${attempt_resp}" | jq -r '.message // empty' 2>/dev/null)" == "created" ]]
}

if run_with_retry "CreateDatabase" attempt_create_database; then
    r_create="PASS"
else
    create_msg=$(echo "${attempt_resp}" | jq -r '.message // empty' 2>/dev/null)
    abort "CreateDatabase" "${create_msg:-unexpected response}"
fi
echo "[${CSP_NAME}] CreateDatabase: ${r_create}"

# ── ListDatabases ─────────────────────────────────────────────────────────────
echo "[${CSP_NAME}] ListDatabases: verifying '${DB_NAME}' is present"

attempt_list_databases() {
    attempt_resp=$(curl -u "${SPIDER_AUTH}" -sX GET \
      "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}/databases?ConnectionName=${CONNECTION_NAME}" \
      -H "X-Master-User-Name: ${MASTER_USER_NAME}" \
      -H "X-Master-User-Password: ${MASTER_USER_PASSWORD}" 2>&1)
    [[ -z "$(echo "${attempt_resp}" | jq -r '.message // empty' 2>/dev/null)" ]]
}

if run_with_retry "ListDatabases" attempt_list_databases; then
    list_resp="${attempt_resp}"
else
    list_err=$(echo "${attempt_resp}" | jq -r '.message // empty' 2>/dev/null)
    abort "ListDatabases" "${list_err:-unexpected response}"
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

attempt_delete_database() {
    attempt_resp=$(curl -u "${SPIDER_AUTH}" -sX DELETE \
      "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}/databases/${DB_NAME}" \
      -H 'Content-Type: application/json' \
      -d "{
        \"ConnectionName\": \"${CONNECTION_NAME}\",
        \"MasterUserName\": \"${MASTER_USER_NAME}\",
        \"MasterUserPassword\": \"${MASTER_USER_PASSWORD}\"
      }" 2>&1)
    [[ "$(echo "${attempt_resp}" | jq -r '.message // empty' 2>/dev/null)" == "deleted" ]]
}

if run_with_retry "DeleteDatabase" attempt_delete_database; then
    r_delete="PASS"
else
    delete_msg=$(echo "${attempt_resp}" | jq -r '.message // empty' 2>/dev/null)
    abort "DeleteDatabase" "${delete_msg:-unexpected response}"
fi
echo "[${CSP_NAME}] DeleteDatabase: ${r_delete}"

# ── ListDatabases (verify deleted) ───────────────────────────────────────────
echo "[${CSP_NAME}] ListDatabases: verifying '${DB_NAME}' is removed"

attempt_verify_databases() {
    attempt_resp=$(curl -u "${SPIDER_AUTH}" -sX GET \
      "${SPIDER_URL}/spider/rdbms/${RDBMS_NAME}/databases?ConnectionName=${CONNECTION_NAME}" \
      -H "X-Master-User-Name: ${MASTER_USER_NAME}" \
      -H "X-Master-User-Password: ${MASTER_USER_PASSWORD}" 2>&1)
    [[ -z "$(echo "${attempt_resp}" | jq -r '.message // empty' 2>/dev/null)" ]]
}

if run_with_retry "ListDatabases(verify)" attempt_verify_databases; then
    verify_resp="${attempt_resp}"
else
    verify_err=$(echo "${attempt_resp}" | jq -r '.message // empty' 2>/dev/null)
    abort "ListDatabases(verify)" "${verify_err:-unexpected response}"
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
