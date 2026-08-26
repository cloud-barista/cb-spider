#!/bin/bash

# CB-Spider VM Default-PublicIP Test - Common Test Script
# Flow: Create VM -> Poll until Running -> record PublicIP -> SSH-login check
#       -> Suspend -> Poll until Suspended -> record PublicIP
#       -> Resume  -> Poll until Running   -> record PublicIP -> SSH-login check
#       -> Compare Initial vs Resumed PublicIP -> Write result file
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   VM_NAME         - VM instance name
#   CREATE_JSON     - JSON body for POST /spider/vm
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL           - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH          - Basic auth credentials (default: admin:****)
#   MAX_WAIT_SEC         - Max seconds to wait for Running after create (default: 1800)
#   POLL_INTERVAL        - Polling interval in seconds (default: 15)
#   SUSPEND_MAX_WAIT_SEC - Max seconds to wait for Suspended (default: 600)
#   RESUME_MAX_WAIT_SEC  - Max seconds to wait for Running after resume (default: 600)
#   KEY_DIR              - Directory holding the PrivateKey saved by
#                           common-network-prepare.sh (default: /tmp/vm_publicip_keys).
#                           The actual file is "${KEY_DIR}/${CONNECTION_NAME}.pem".
#   SSH_USER             - OS login user for the SSH-login check (default: cb-user,
#                           CB-Spider's default cloud-init account on Linux images)
#   SSH_MAX_ATTEMPTS     - SSH connection retry count (default: 10)
#   SSH_RETRY_INTERVAL   - Seconds between SSH retries (default: 30)

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
MAX_WAIT_SEC="${MAX_WAIT_SEC:-1800}"
POLL_INTERVAL="${POLL_INTERVAL:-15}"
SUSPEND_MAX_WAIT_SEC="${SUSPEND_MAX_WAIT_SEC:-600}"
RESUME_MAX_WAIT_SEC="${RESUME_MAX_WAIT_SEC:-600}"

# Same derivation as common-network-prepare.sh: both scripts compute this path
# independently from CONNECTION_NAME alone, no explicit hand-off needed.
KEY_DIR="${KEY_DIR:-/tmp/vm_publicip_keys}"
KEY_FILE="${KEY_DIR}/${CONNECTION_NAME}.pem"
SSH_USER="${SSH_USER:-cb-user}"
SSH_MAX_ATTEMPTS="${SSH_MAX_ATTEMPTS:-10}"
SSH_RETRY_INTERVAL="${SSH_RETRY_INTERVAL:-30}"

mkdir -p "$(dirname "${RESULT_FILE}")"

# write_fail_result CODE
# Writes a result line with the given failure code in the Status column and
# exits 1. Keeps the pipe-field count consistent so the summary table parses.
write_fail_result() {
    echo "${CSP_NAME}|$1|-|-|-|-|-|-|-" > "${RESULT_FILE}"
    exit 1
}

# get_status -> prints current VMStatus (e.g. Running, Suspended, ...) or "unknown"
get_status() {
    local resp
    resp=$(curl -u "${SPIDER_AUTH}" -s \
      "${SPIDER_URL}/spider/vmstatus/${VM_NAME}?ConnectionName=${CONNECTION_NAME}" 2>&1)
    echo "${resp}" | jq -r '.Status // "unknown"' 2>/dev/null
}

# get_public_ip -> prints current PublicIP field of GET /spider/vm/{Name} (may be empty)
get_public_ip() {
    local resp
    resp=$(curl -u "${SPIDER_AUTH}" -s \
      "${SPIDER_URL}/spider/vm/${VM_NAME}?ConnectionName=${CONNECTION_NAME}" 2>&1)
    echo "${resp}" | jq -r '.PublicIP // empty' 2>/dev/null
}

# wait_for_status TARGET_STATUS MAX_WAIT PHASE_LABEL
# Polls get_status every POLL_INTERVAL seconds (case-insensitive match) until
# it equals TARGET_STATUS, or MAX_WAIT is exceeded (returns 1 on timeout).
wait_for_status() {
    local target="$1" max_wait="$2" phase="$3"
    local target_lower
    target_lower=$(echo "${target}" | tr '[:upper:]' '[:lower:]')

    local elapsed=0
    while true; do
        sleep "${POLL_INTERVAL}"
        elapsed=$((elapsed + POLL_INTERVAL))

        local cur_status status_lower
        cur_status=$(get_status)
        status_lower=$(echo "${cur_status}" | tr '[:upper:]' '[:lower:]')
        echo "[${CSP_NAME}] [${phase}] Status: ${cur_status} (elapsed: ${elapsed}s)"

        if [[ "${status_lower}" == "${target_lower}" ]]; then
            return 0
        fi
        if [[ "${status_lower}" == "failed" ]]; then
            echo "[${CSP_NAME}] [${phase}] VM entered Failed state."
            return 1
        fi
        if [[ ${elapsed} -ge ${max_wait} ]]; then
            echo "[${CSP_NAME}] [${phase}] TIMEOUT: expected '${target}' within ${max_wait}s, last status '${cur_status}'"
            return 1
        fi
    done
}

# check_ssh IP PHASE_LABEL
# Prints one of: OK | FAIL | NO_IP | NO_KEY
# Retries the SSH login up to SSH_MAX_ATTEMPTS times (cloud-init needs time
# to provision the SSH_USER account after the VM/PublicIP becomes reachable).
check_ssh() {
    local ip="$1" phase="$2"

    if [[ -z "${ip}" || "${ip}" == "(none)" ]]; then
        echo "[${CSP_NAME}] [${phase}] No PublicIP available — skipping SSH check." >&2
        echo "NO_IP"
        return
    fi
    if [[ ! -f "${KEY_FILE}" ]]; then
        echo "[${CSP_NAME}] [${phase}] No PrivateKey file at ${KEY_FILE} — skipping SSH check." >&2
        echo "NO_KEY"
        return
    fi

    local attempt
    for ((attempt = 1; attempt <= SSH_MAX_ATTEMPTS; attempt++)); do
        echo "[${CSP_NAME}] [${phase}] SSH attempt ${attempt}/${SSH_MAX_ATTEMPTS} -> ${SSH_USER}@${ip}" >&2
        if ssh -i "${KEY_FILE}" \
             -o StrictHostKeyChecking=no \
             -o UserKnownHostsFile=/dev/null \
             -o ConnectTimeout=10 \
             -o BatchMode=yes \
             -o LogLevel=ERROR \
             "${SSH_USER}@${ip}" "echo cb-spider-ssh-ok" 2>/dev/null | grep -q "cb-spider-ssh-ok"; then
            echo "[${CSP_NAME}] [${phase}] SSH login OK (attempt ${attempt})" >&2
            echo "OK"
            return
        fi
        [[ ${attempt} -lt ${SSH_MAX_ATTEMPTS} ]] && sleep "${SSH_RETRY_INTERVAL}"
    done

    echo "[${CSP_NAME}] [${phase}] SSH login FAILED after ${SSH_MAX_ATTEMPTS} attempts." >&2
    echo "FAIL"
}

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

echo "[${CSP_NAME}] [${timestamp}] Creating VM '${VM_NAME}'..."

# ── 1) Create VM ──────────────────────────────────────────────────────────────
create_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/vm" \
  -H 'Content-Type: application/json' \
  -d "${CREATE_JSON}" 2>&1)

err_msg=$(echo "${create_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${err_msg}" ]]; then
    echo "[${CSP_NAME}] ERROR on create: ${err_msg}"
    write_fail_result "CREATE_ERROR"
fi
echo "[${CSP_NAME}] Create request accepted."

# ── 2) Wait for Running ───────────────────────────────────────────────────────
echo "[${CSP_NAME}] Waiting for VM to become Running (poll every ${POLL_INTERVAL}s, max ${MAX_WAIT_SEC}s)..."
if ! wait_for_status "Running" "${MAX_WAIT_SEC}" "create"; then
    write_fail_result "CREATE_TIMEOUT_OR_FAILED"
fi
echo "[${CSP_NAME}] VM is Running."

# ── 3) Record initial PublicIP, then SSH-login check ─────────────────────────
ip_initial=$(get_public_ip)
[[ -z "${ip_initial}" ]] && ip_initial="(none)"
echo "[${CSP_NAME}] PublicIP after create: ${ip_initial}"

ssh_initial=$(check_ssh "${ip_initial}" "create")
echo "[${CSP_NAME}] SSH check after create: ${ssh_initial}"

# ── 4) Suspend, then record PublicIP ─────────────────────────────────────────
echo "[${CSP_NAME}] Sending suspend action..."
suspend_resp=$(curl -u "${SPIDER_AUTH}" -s \
  "${SPIDER_URL}/spider/controlvm/${VM_NAME}?ConnectionName=${CONNECTION_NAME}&action=suspend" 2>&1)
suspend_err=$(echo "${suspend_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${suspend_err}" ]]; then
    echo "[${CSP_NAME}] ERROR on suspend: ${suspend_err}"
    write_fail_result "SUSPEND_ERROR"
fi

echo "[${CSP_NAME}] Waiting for VM to become Suspended (max ${SUSPEND_MAX_WAIT_SEC}s)..."
if ! wait_for_status "Suspended" "${SUSPEND_MAX_WAIT_SEC}" "suspend"; then
    write_fail_result "SUSPEND_TIMEOUT_OR_FAILED"
fi
echo "[${CSP_NAME}] VM is Suspended."

ip_suspended=$(get_public_ip)
[[ -z "${ip_suspended}" ]] && ip_suspended="(none)"
echo "[${CSP_NAME}] PublicIP while Suspended: ${ip_suspended}"

# ── 5) Resume, then record PublicIP, then SSH-login check ───────────────────
echo "[${CSP_NAME}] Sending resume action..."
resume_resp=$(curl -u "${SPIDER_AUTH}" -s \
  "${SPIDER_URL}/spider/controlvm/${VM_NAME}?ConnectionName=${CONNECTION_NAME}&action=resume" 2>&1)
resume_err=$(echo "${resume_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${resume_err}" ]]; then
    echo "[${CSP_NAME}] ERROR on resume: ${resume_err}"
    write_fail_result "RESUME_ERROR"
fi

echo "[${CSP_NAME}] Waiting for VM to become Running again (max ${RESUME_MAX_WAIT_SEC}s)..."
if ! wait_for_status "Running" "${RESUME_MAX_WAIT_SEC}" "resume"; then
    write_fail_result "RESUME_TIMEOUT_OR_FAILED"
fi
echo "[${CSP_NAME}] VM is Running again."

ip_resumed=$(get_public_ip)
[[ -z "${ip_resumed}" ]] && ip_resumed="(none)"
echo "[${CSP_NAME}] PublicIP after resume: ${ip_resumed}"

ssh_resumed=$(check_ssh "${ip_resumed}" "resume")
echo "[${CSP_NAME}] SSH check after resume: ${ssh_resumed}"

# ── 6) Compare Initial vs Resumed PublicIP ───────────────────────────────────
if [[ "${ip_initial}" == "(none)" && "${ip_resumed}" == "(none)" ]]; then
    comparison="NONE_BOTH"
elif [[ "${ip_initial}" == "(none)" || "${ip_resumed}" == "(none)" ]]; then
    comparison="CHANGED_TO_NONE"
elif [[ "${ip_initial}" == "${ip_resumed}" ]]; then
    comparison="SAME"
else
    comparison="CHANGED"
fi

end_time=$(date +%s)
elapsed_total=$((end_time - start_time))
elapsed_fmt=$(format_elapsed "${elapsed_total}")

echo "[${CSP_NAME}] Done. Initial=${ip_initial}(ssh:${ssh_initial}) Suspended=${ip_suspended} Resumed=${ip_resumed}(ssh:${ssh_resumed}) (${comparison}) (total elapsed: ${elapsed_fmt})"

# Format: CSP|Status|InitialIP|SuspendedIP|ResumedIP|Comparison|SSHInitial|SSHResumed|Elapsed
echo "${CSP_NAME}|OK|${ip_initial}|${ip_suspended}|${ip_resumed}|${comparison}|${ssh_initial}|${ssh_resumed}|${elapsed_fmt}" \
  > "${RESULT_FILE}"
