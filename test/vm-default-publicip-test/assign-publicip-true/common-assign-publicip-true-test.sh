#!/bin/bash

# CB-Spider VM AssignPublicIP=true Test - Common Test Script
# Flow: Create VM with ReqInfo.AssignPublicIP=true -> Poll until Running ->
#       Get VM Info -> Verify PrivateIP AND PublicIP are both set -> SSH check
#       (initial) -> UnassignVMDefaultPublicIP -> Verify PublicIP is now empty
#       -> AssignVMDefaultPublicIP -> Verify PublicIP is set again -> SSH check
#       (re-assigned) -> Write result
# Author: CB-Spider Team
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME        - Display name (e.g., AWS)
#   CONNECTION_NAME - Spider connection config name
#   VM_NAME         - VM instance name
#   CREATE_JSON     - JSON body for POST /spider/vm (must include "AssignPublicIP": true)
#   RESULT_FILE     - Path to write pipe-separated result line
#
# Optional env vars:
#   SPIDER_URL           - Spider REST API URL (default: http://localhost:1024)
#   SPIDER_AUTH          - Basic auth credentials (default: admin:****)
#   MAX_WAIT_SEC         - Max seconds to wait for Running after create (default: 1800)
#   POLL_INTERVAL        - Polling interval in seconds (default: 15)
#   KEY_DIR              - Directory holding the PrivateKey saved by
#                           ../common-network-prepare.sh (default: /tmp/vm_publicip_keys).
#                           The actual file is "${KEY_DIR}/${CONNECTION_NAME}.pem".
#   SSH_USER             - OS login user for the SSH-login check (default: cb-user)
#   SSH_MAX_ATTEMPTS     - SSH connection retry count (default: 10)
#   SSH_RETRY_INTERVAL   - Seconds between SSH retries (default: 30)
#
# Result file format (11 fields):
#   CSP|Result|VMStatus|PrivateIP|PublicIP(Initial)|SSH(Initial)|PublicIP(Unassigned)|PublicIP(Reassigned)|SSH(Reassigned)|Elapsed|Reason

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

KEY_DIR="${KEY_DIR:-/tmp/vm_publicip_keys}"
KEY_FILE="${KEY_DIR}/${CONNECTION_NAME}.pem"
SSH_USER="${SSH_USER:-cb-user}"
SSH_MAX_ATTEMPTS="${SSH_MAX_ATTEMPTS:-10}"
SSH_RETRY_INTERVAL="${SSH_RETRY_INTERVAL:-30}"

mkdir -p "$(dirname "${RESULT_FILE}")"

# write_fail_result STATUS REASON
# Writes a FAIL result line and exits 1. Keeps the pipe-field count consistent
# (11 fields) so the summary table parses correctly. Fields already captured
# before the failing step (e.g. PrivateIP/initial PublicIP/SSH when a later
# step like Unassign fails) are preserved instead of being blanked to "-",
# so a FAIL row doesn't misleadingly look like VM creation itself failed.
write_fail_result() {
    echo "${CSP_NAME}|FAIL|$1|${private_ip:--}|${initial_public_ip_display:--}|${ssh_initial:--}|${unassigned_public_ip_display:--}|${reassigned_public_ip_display:--}|${ssh_reassigned:--}|${elapsed_fmt:--}|$2" > "${RESULT_FILE}"
    exit 1
}

# get_status -> prints current VMStatus (e.g. Running, Creating, ...) or "unknown"
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

# check_ssh IP PHASE_LABEL -> prints one of: OK | FAIL | NO_IP | NO_KEY
# Retries the SSH login up to SSH_MAX_ATTEMPTS times (cloud-init needs time to
# provision the SSH_USER account after the PublicIP becomes reachable).
check_ssh() {
    local ip="$1" phase="$2"

    if [[ -z "${ip}" || "${ip}" == "(none)" ]]; then
        echo "[${CSP_NAME}] [${phase}] No PublicIP available - skipping SSH check." >&2
        echo "NO_IP"
        return
    fi
    if [[ ! -f "${KEY_FILE}" ]]; then
        echo "[${CSP_NAME}] [${phase}] No PrivateKey file at ${KEY_FILE} - skipping SSH check." >&2
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
        [[ ${attempt} -lt SSH_MAX_ATTEMPTS ]] && sleep "${SSH_RETRY_INTERVAL}"
    done

    echo "[${CSP_NAME}] [${phase}] SSH login FAILED after ${SSH_MAX_ATTEMPTS} attempts." >&2
    echo "FAIL"
}

start_time=$(date +%s)
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

echo "[${CSP_NAME}] [${timestamp}] Creating VM '${VM_NAME}' with AssignPublicIP=true..."

# ── 1) Create VM ──────────────────────────────────────────────────────────────
create_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/vm" \
  -H 'Content-Type: application/json' \
  -d "${CREATE_JSON}" 2>&1)

err_msg=$(echo "${create_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${err_msg}" ]]; then
    echo "[${CSP_NAME}] ERROR on create: ${err_msg}"
    write_fail_result "CREATE_ERROR" "${err_msg}"
fi
echo "[${CSP_NAME}] Create request accepted."

# ── 2) Wait for Running ───────────────────────────────────────────────────────
echo "[${CSP_NAME}] Waiting for VM to become Running (poll every ${POLL_INTERVAL}s, max ${MAX_WAIT_SEC}s)..."

elapsed=0
final_status="unknown"
while true; do
    sleep "${POLL_INTERVAL}"
    elapsed=$((elapsed + POLL_INTERVAL))

    cur_status=$(get_status)
    status_lower=$(echo "${cur_status}" | tr '[:upper:]' '[:lower:]')
    echo "[${CSP_NAME}] Status: ${cur_status} (elapsed: ${elapsed}s)"

    if [[ "${status_lower}" == "running" ]]; then
        final_status="${cur_status}"
        break
    fi
    if [[ "${status_lower}" == "failed" ]]; then
        elapsed_fmt=$(format_elapsed "${elapsed}")
        echo "[${CSP_NAME}] VM entered Failed state."
        write_fail_result "${cur_status}" "VM entered Failed state instead of Running"
    fi
    if [[ ${elapsed} -ge ${MAX_WAIT_SEC} ]]; then
        elapsed_fmt=$(format_elapsed "${elapsed}")
        echo "[${CSP_NAME}] TIMEOUT: VM did not reach Running within ${MAX_WAIT_SEC}s (last status '${cur_status}')"
        write_fail_result "${cur_status}" "Timed out waiting for Running"
    fi
done
echo "[${CSP_NAME}] VM is Running."

# ── 3) Get VM Info, check PrivateIP / PublicIP ───────────────────────────────
info=$(curl -u "${SPIDER_AUTH}" -s \
  "${SPIDER_URL}/spider/vm/${VM_NAME}?ConnectionName=${CONNECTION_NAME}" 2>&1)

private_ip=$(echo "${info}" | jq -r '.PrivateIP // empty' 2>/dev/null)
initial_public_ip=$(echo "${info}" | jq -r '.PublicIP // empty' 2>/dev/null)

private_ip_display="${private_ip:-(none)}"
initial_public_ip_display="${initial_public_ip:-(none)}"

echo "[${CSP_NAME}] PrivateIP=${private_ip_display}  PublicIP=${initial_public_ip_display}"

# ── 4) Verify: must have a PrivateIP AND must have a PublicIP ───────────────
if [[ -z "${private_ip}" ]]; then
    echo "[${CSP_NAME}] FAIL: PrivateIP is empty."
    write_fail_result "${final_status}" "PrivateIP is empty"
fi

if [[ -z "${initial_public_ip}" ]]; then
    echo "[${CSP_NAME}] FAIL: PublicIP was not assigned even though AssignPublicIP=true was requested."
    write_fail_result "${final_status}" "PublicIP unexpectedly absent"
fi

echo "[${CSP_NAME}] OK: VM Running with PrivateIP=${private_ip} and PublicIP=${initial_public_ip}."

# ── 5) SSH check via the initial PublicIP ────────────────────────────────────
ssh_initial=$(check_ssh "${initial_public_ip}" "initial")
echo "[${CSP_NAME}] SSH check (initial) result: ${ssh_initial}"

if [[ "${ssh_initial}" != "OK" ]]; then
    write_fail_result "${final_status}" "SSH check after create did not succeed: ${ssh_initial}"
fi

# ── 6) UnassignVMDefaultPublicIP ─────────────────────────────────────────────
echo "[${CSP_NAME}] Requesting UnassignVMDefaultPublicIP..."
unassign_resp=$(curl -u "${SPIDER_AUTH}" -sX DELETE "${SPIDER_URL}/spider/vm/${VM_NAME}/publicip" \
  -H 'Content-Type: application/json' \
  -d "{\"ConnectionName\": \"${CONNECTION_NAME}\"}" 2>&1)

unassign_err=$(echo "${unassign_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${unassign_err}" ]]; then
    echo "[${CSP_NAME}] ERROR on UnassignVMDefaultPublicIP: ${unassign_err}"
    write_fail_result "${final_status}" "UnassignVMDefaultPublicIP error: ${unassign_err}"
fi
unassign_result=$(echo "${unassign_resp}" | jq -r '.Result // empty' 2>/dev/null)
if [[ "${unassign_result}" != "true" ]]; then
    echo "[${CSP_NAME}] FAIL: UnassignVMDefaultPublicIP did not return Result=true (got '${unassign_result}')."
    write_fail_result "${final_status}" "UnassignVMDefaultPublicIP returned Result=${unassign_result}"
fi

# ── 7) Confirm PublicIP is no longer assigned ────────────────────────────────
unassigned_public_ip=$(get_public_ip)
unassigned_public_ip_display="${unassigned_public_ip:-(none)}"
echo "[${CSP_NAME}] PublicIP after UnassignVMDefaultPublicIP: ${unassigned_public_ip_display}"

if [[ -n "${unassigned_public_ip}" ]]; then
    echo "[${CSP_NAME}] FAIL: PublicIP still present after UnassignVMDefaultPublicIP."
    write_fail_result "${final_status}" "PublicIP unexpectedly present after Unassign: ${unassigned_public_ip}"
fi

# ── 8) AssignVMDefaultPublicIP ────────────────────────────────────────────────
echo "[${CSP_NAME}] Requesting AssignVMDefaultPublicIP..."
assign_resp=$(curl -u "${SPIDER_AUTH}" -sX POST "${SPIDER_URL}/spider/vm/${VM_NAME}/publicip" \
  -H 'Content-Type: application/json' \
  -d "{\"ConnectionName\": \"${CONNECTION_NAME}\"}" 2>&1)

assign_err=$(echo "${assign_resp}" | jq -r '.message // empty' 2>/dev/null)
if [[ -n "${assign_err}" ]]; then
    echo "[${CSP_NAME}] ERROR on AssignVMDefaultPublicIP: ${assign_err}"
    write_fail_result "${final_status}" "AssignVMDefaultPublicIP error: ${assign_err}"
fi

# ── 9) Confirm PublicIP was re-assigned ──────────────────────────────────────
reassigned_public_ip=$(get_public_ip)
reassigned_public_ip_display="${reassigned_public_ip:-(none)}"
echo "[${CSP_NAME}] PublicIP after AssignVMDefaultPublicIP: ${reassigned_public_ip_display}"

if [[ -z "${reassigned_public_ip}" ]]; then
    echo "[${CSP_NAME}] FAIL: PublicIP was not assigned after AssignVMDefaultPublicIP."
    write_fail_result "${final_status}" "AssignVMDefaultPublicIP did not result in a PublicIP"
fi

# ── 10) SSH check via the re-assigned PublicIP ───────────────────────────────
ssh_reassigned=$(check_ssh "${reassigned_public_ip}" "reassigned")
echo "[${CSP_NAME}] SSH check (reassigned) result: ${ssh_reassigned}"

# Anything other than a confirmed OK means SSH reachability couldn't be
# verified (including NO_KEY - a missing PrivateKey file leaves the result
# unknown, not passing), so it fails the test.
if [[ "${ssh_reassigned}" != "OK" ]]; then
    write_fail_result "${final_status}" "SSH check after AssignVMDefaultPublicIP did not succeed: ${ssh_reassigned}"
fi

end_time=$(date +%s)
elapsed_total=$((end_time - start_time))
elapsed_fmt=$(format_elapsed "${elapsed_total}")

echo "[${CSP_NAME}] PASS: full AssignPublicIP=true + Unassign/Assign lifecycle succeeded (${elapsed_fmt})"

# Format: CSP|Result|VMStatus|PrivateIP|PublicIP(Initial)|SSH(Initial)|PublicIP(Unassigned)|PublicIP(Reassigned)|SSH(Reassigned)|Elapsed|Reason
echo "${CSP_NAME}|PASS|${final_status}|${private_ip}|${initial_public_ip_display}|${ssh_initial}|${unassigned_public_ip_display}|${reassigned_public_ip_display}|${ssh_reassigned}|${elapsed_fmt}|-" \
  > "${RESULT_FILE}"
