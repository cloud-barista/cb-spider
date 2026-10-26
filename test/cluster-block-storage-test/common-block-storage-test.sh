#!/bin/bash

# CB-Spider Cluster Block Storage Driver (Block-PV/PVC) Test - Common Script
# Runs every test step for one CSP in batch mode and writes a result line.
# Requires: curl, jq, kubectl, ~/.cb-spider/.spider-credential (kubeconfig exec token)
# Author: CB-Spider Team
#
# Steps (result index):
#   0 Test Cluster Create      : VPC/SG/KeyPair -> Cluster (+NodeGroup) -> Active -> Nodes Ready
#   1 Driver Enable            : PUT /cluster/{Name}/storagedriver -> List Status Enabled
#   2 CSI Driver Installed     : CSIDriver object exists and is registered on CSINode
#   3 StorageClass             : CSP-provided SC of the driver, or create one
#   4 PVC -> PV -> Bound       : PV created, CSP disk exists (Spider ListAllDisk)
#   5 Attach/Mount, R/W        : write/read, Pod re-create keeps data
#   6 PVC Delete               : PV deleted, CSP disk deleted (Spider ListAllDisk)
#   7 Driver Disable           : Result true, Status Disabled, CSI driver gone from CSINode
#                                (EXPECT_DISABLE=reject or BuiltIn: rejection expected)
#   8 Driver Re-Enable         : Status Enabled and CSI driver registered again
#
# Required env vars (set by per-CSP scripts):
#   CSP_NAME, CSP_KEY, CONNECTION_NAME, VPC_CIDR, SUBNET_JSON, CLUSTER_TYPE (I|II),
#   NODE_VM_SPEC
# Optional env vars:
#   NODE_IMAGE, NODE_ROOT_DISK_TYPE, NODE_ROOT_DISK_SIZE, K8S_VERSION, DRIVER_VERSION,
#   EXPECT_DISABLE (ok|reject, default ok), SC_PARAMETERS_JSON (default {}),
#   SG_RULES_JSON, NODE_SIZE (default 2), TEST_IMAGE (default busybox:1.36),
#   KEEP_CLUSTER (true: skip cleanup), SPIDER_URL, SPIDER_AUTH, RESULT_DIR,
#   CLUSTER_TIMEOUT (default 3600s), DRIVER_TIMEOUT (900s), K8S_TIMEOUT (600s)
#
# On SIGINT/SIGTERM the result is written and cleanup runs; a second signal aborts cleanup.

SPIDER_URL="${SPIDER_URL:-http://localhost:1024}"
SPIDER_AUTH="${SPIDER_AUTH:-admin:****}"
RESULT_DIR="${RESULT_DIR:-/tmp/cluster_block_storage_results}"
CLUSTER_NAME="${CLUSTER_NAME:-blk-test-${CSP_KEY}}"
VPC_NAME="${VPC_NAME:-blk-test-vpc}"
SG_NAME="${SG_NAME:-blk-test-sg}"
KEYPAIR_NAME="${KEYPAIR_NAME:-blk-test-key}"
NODEGROUP_NAME="${NODEGROUP_NAME:-ng-01}"
NODE_SIZE="${NODE_SIZE:-2}"
EXPECT_DISABLE="${EXPECT_DISABLE:-ok}"
TEST_IMAGE="${TEST_IMAGE:-busybox:1.36}"
CLUSTER_TIMEOUT="${CLUSTER_TIMEOUT:-3600}"
DRIVER_TIMEOUT="${DRIVER_TIMEOUT:-900}"
K8S_TIMEOUT="${K8S_TIMEOUT:-600}"
TEST_SC_NAME="spider-blk-test-sc"
PVC_NAME="spider-blk-test-pvc"
POD_NAME="spider-blk-test-pod"
SPIDER_CREDENTIAL="${HOME}/.cb-spider/.spider-credential"

# Note: not using "${VAR:-...}" for JSON defaults — bash ends the expansion at the
# first unescaped '}', so a default containing JSON braces gets truncated.
if [[ -z "${SG_RULES_JSON}" ]]; then
    SG_RULES_JSON="[{\"Direction\":\"inbound\",\"IPProtocol\":\"ALL\",\"FromPort\":\"-1\",\"ToPort\":\"-1\",\"CIDR\":\"${VPC_CIDR}\"},
                    {\"Direction\":\"inbound\",\"IPProtocol\":\"TCP\",\"FromPort\":\"443\",\"ToPort\":\"443\",\"CIDR\":\"0.0.0.0/0\"}]"
fi
if [[ -z "${SC_PARAMETERS_JSON}" ]]; then
    SC_PARAMETERS_JSON='{}'
fi

WORK_DIR="${RESULT_DIR}/${CSP_KEY}"
LOG_FILE="${WORK_DIR}/test.log"
RESULT_FILE="${RESULT_DIR}/result_${CSP_KEY}.txt"
export KUBECONFIG="${WORK_DIR}/kubeconfig"
mkdir -p "${WORK_DIR}"
: > "${LOG_FILE}"

# Result marks: ✅ pass, ❌ fail, ➖ not applicable (expected), ⏸ not run
R=("⏸" "⏸" "⏸" "⏸" "⏸" "⏸" "⏸" "⏸" "⏸")
NOTES=()

log()  { echo "[${CSP_NAME}] [$(date '+%H:%M:%S')] $*" | tee -a "${LOG_FILE}"; }
note() { NOTES+=("$*"); log "NOTE: $*"; }
pass() { R[$1]="✅"; log "STEP $1 PASS ${2:-}"; }
fail() { R[$1]="❌"; note "S$1: $2"; }

# spider METHOD PATH [JSON] -> sets HTTP_CODE and BODY
spider() {
    local out
    out=$(curl -u "${SPIDER_AUTH}" -s -w $'\n%{http_code}' -X "$1" "${SPIDER_URL}/spider$2" \
          -H 'Content-Type: application/json' ${3:+-d "$3"} 2>>"${LOG_FILE}")
    HTTP_CODE="${out##*$'\n'}"
    BODY="${out%$'\n'*}"
    [[ "${HTTP_CODE}" == "000" ]] && BODY="Spider unreachable (${SPIDER_URL})"
    echo "$1 $2 -> ${HTTP_CODE} ${BODY:0:500}" >> "${LOG_FILE}"
}

# wait_for TIMEOUT INTERVAL FUNC: retries FUNC until it returns 0.
# FUNC returns 3 for a terminal state, which stops waiting early (wait_for returns 3).
wait_for() {
    local end=$(( SECONDS + $1 )) interval=$2 fn=$3 rc
    while (( SECONDS < end )); do
        "${fn}"; rc=$?
        (( rc == 0 || rc == 3 )) && return ${rc}
        sleep "${interval}"
    done
    return 1
}

write_result() {
    local note_str="" n
    for n in "${NOTES[@]}"; do note_str+="${note_str:+; }${n//|//}"; done
    (IFS='|'; echo "${CSP_NAME}|${R[*]}|${note_str}") > "${RESULT_FILE}"
    log "Result: ${R[*]}"
}

conn_body() { echo "{\"ConnectionName\":\"${CONNECTION_NAME}\"}"; }

# ── helpers for checks ───────────────────────────────────────────────────────
# Cluster/NodeGroup "Inactive" is not treated as terminal: several drivers map
# unknown or transient CSP states to it. The last status goes into the failure note.
cluster_active() {
    spider GET "/cluster/${CLUSTER_NAME}?ConnectionName=${CONNECTION_NAME}"
    CLUSTER_STATUS=$(echo "${BODY}" | jq -r '.Status // empty' 2>/dev/null)
    [[ "${CLUSTER_STATUS}" == "Active" ]]
}
nodegroups_active() {
    spider GET "/cluster/${CLUSTER_NAME}?ConnectionName=${CONNECTION_NAME}"
    local total active
    NODEGROUP_STATUS=$(echo "${BODY}" | jq -r '[.NodeGroupList[]?.Status] | join(",")' 2>/dev/null)
    total=$(echo "${BODY}" | jq '[.NodeGroupList[]?] | length')
    active=$(echo "${BODY}" | jq '[.NodeGroupList[]? | select(.Status=="Active")] | length')
    (( total > 0 && total == active ))
}
nodes_ready() {
    local n
    n=$(kubectl get nodes --no-headers 2>>"${LOG_FILE}" | awk '$2=="Ready"' | wc -l)
    (( n >= 1 ))
}
# Sets DRIVER_JSON / DRIVER_STATUS (not called in a subshell, so the globals survive)
driver_status() {
    spider GET "/cluster/${CLUSTER_NAME}/storagedriver?ConnectionName=${CONNECTION_NAME}"
    DRIVER_JSON=$(echo "${BODY}" | jq -c '.storagedriver[]? | select(.Kind=="Block")' 2>/dev/null)
    DRIVER_STATUS=$(echo "${DRIVER_JSON}" | jq -r '.Status // empty' 2>/dev/null)
}
# "Failed" can be transient (e.g., add-on Degraded while its Pods start), so it is
# terminal only after it persists for 4 polls in a row. Reset FAILED_POLLS before waiting.
driver_enabled() {
    driver_status
    [[ "${DRIVER_STATUS}" == "Enabled" ]] && return 0
    if [[ "${DRIVER_STATUS}" == "Failed" ]]; then
        (( ++FAILED_POLLS >= 4 )) && return 3
    else
        FAILED_POLLS=0
    fi
    return 1
}
driver_disabled() { driver_status; [[ "${DRIVER_STATUS}" == "Disabled" ]]; }
csi_registered() {
    kubectl get csidriver "${CSI_NAME}" >/dev/null 2>>"${LOG_FILE}" || return 1
    local n
    n=$(kubectl get csinode -o json 2>>"${LOG_FILE}" \
        | jq --arg d "${CSI_NAME}" '[.items[] | select([.spec.drivers[]?.name] | index($d))] | length')
    (( n >= 1 ))
}
csi_deregistered() { ! csi_registered; }
pvc_bound() { [[ "$(kubectl get pvc "${PVC_NAME}" -o jsonpath='{.status.phase}' 2>/dev/null)" == "Bound" ]]; }
pod_ready() { kubectl wait --for=condition=Ready "pod/${POD_NAME}" --timeout=10s >/dev/null 2>&1; }
pv_gone()   { ! kubectl get pv "${PV_NAME}" >/dev/null 2>&1; }

# CSP disk lookup through Spider (no CSP console): matches the PV volumeHandle
# against SystemIds of all CSP disks, in either direction (handles may be full paths).
csp_disk_exists() {
    spider GET "/alldisk?ConnectionName=${CONNECTION_NAME}"
    [[ "${HTTP_CODE}" == "200" ]] || return 2
    echo "${BODY}" | jq -e --arg h "${VOLUME_HANDLE}" '
        [.AllList.MappedList[]?, .AllList.OnlyCSPList[]? | .SystemId // empty | ascii_downcase]
        | any(. as $s | ($h | ascii_downcase) as $hl | ($hl | contains($s)) or ($s | contains($hl)))' >/dev/null
}
csp_disk_gone() { csp_disk_exists; [[ $? -eq 1 ]]; }

apply_pod() {
    kubectl apply -f - >>"${LOG_FILE}" 2>&1 <<EOF
apiVersion: v1
kind: Pod
metadata: { name: ${POD_NAME} }
spec:
  containers:
  - name: app
    image: ${TEST_IMAGE}
    command: ["sh", "-c", "sleep 36000"]
    volumeMounts: [{ name: vol, mountPath: /data }]
  volumes:
  - name: vol
    persistentVolumeClaim: { claimName: ${PVC_NAME} }
EOF
}

# Deletes the test Pod/PVC and waits until the PV and the CSP disk are released.
# Must run while the driver is installed: without it the CSP disk is never deleted.
delete_test_volume() {
    PV_NAME="${PV_NAME:-$(kubectl get pvc "${PVC_NAME}" -o jsonpath='{.spec.volumeName}' 2>/dev/null)}"
    if [[ -n "${PV_NAME}" && -z "${VOLUME_HANDLE}" ]]; then
        VOLUME_HANDLE=$(kubectl get pv "${PV_NAME}" -o jsonpath='{.spec.csi.volumeHandle}' 2>/dev/null)
    fi
    kubectl delete pod "${POD_NAME}" --ignore-not-found --wait=true --timeout=300s >>"${LOG_FILE}" 2>&1
    kubectl delete pvc "${PVC_NAME}" --ignore-not-found --wait=true --timeout=300s >>"${LOG_FILE}" 2>&1
    [[ -z "${PV_NAME}" ]] && return 0
    wait_for "${K8S_TIMEOUT}" 10 pv_gone || return 1
    [[ -z "${VOLUME_HANDLE}" ]] && return 0
    wait_for "${K8S_TIMEOUT}" 15 csp_disk_gone || return 2
}

cleanup() {
    if [[ "${KEEP_CLUSTER}" == "true" ]]; then
        note "cleanup skipped (KEEP_CLUSTER=true)"
        return
    fi
    log "Cleanup: k8s objects, cluster, network"
    if [[ -s "${KUBECONFIG}" ]]; then
        delete_test_volume || note "cleanup: test volume not released (PV ${PV_NAME})"
    fi
    spider DELETE "/cluster/${CLUSTER_NAME}" "$(conn_body)"
    cluster_gone() { spider GET "/cluster/${CLUSTER_NAME}?ConnectionName=${CONNECTION_NAME}"; [[ "${HTTP_CODE}" != "200" ]]; }
    wait_for "${CLUSTER_TIMEOUT}" 30 cluster_gone || note "cleanup: cluster delete timeout"
    for res in "securitygroup/${SG_NAME}" "keypair/${KEYPAIR_NAME}" "vpc/${VPC_NAME}"; do
        delete_res() { spider DELETE "/${res}" "$(conn_body)"; [[ "${HTTP_CODE}" == "200" ]]; }
        wait_for 600 30 delete_res || note "cleanup: ${res} delete failed"
    done
    # Cluster deletion does not delete CSI-provisioned disks; report any left behind
    if [[ -n "${VOLUME_HANDLE}" ]] && csp_disk_exists; then
        note "cleanup: CSP disk left, delete it manually (${VOLUME_HANDLE})"
    fi
}

on_signal() {
    trap - INT TERM    # a second signal aborts cleanup
    note "interrupted by signal"
    write_result
    cleanup
    write_result
    exit 130
}

# ═════════════════════════════════════════════════════════════════════════════
start_time=$(date +%s)
log "Start (Cluster: ${CLUSTER_NAME}, Connection: ${CONNECTION_NAME})"

for v in CSP_NAME CSP_KEY CONNECTION_NAME VPC_CIDR SUBNET_JSON CLUSTER_TYPE NODE_VM_SPEC; do
    if [[ -z "${!v}" || "${!v}" == *SET_ME* ]]; then
        R[0]="❌"; note "required env ${v} is not set"
        write_result; exit 1
    fi
done
for cmd in curl jq kubectl; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
        R[0]="❌"; note "required command ${cmd} not found"
        write_result; exit 1
    fi
done

trap on_signal INT TERM

# ── Step 0: Test Cluster Create ──────────────────────────────────────────────
step0() {
    spider GET "/vpc/${VPC_NAME}?ConnectionName=${CONNECTION_NAME}"
    if [[ "${HTTP_CODE}" != "200" ]]; then
        spider POST "/vpc" "{\"ConnectionName\":\"${CONNECTION_NAME}\",\"ReqInfo\":{\"Name\":\"${VPC_NAME}\",
            \"IPv4_CIDR\":\"${VPC_CIDR}\",\"SubnetInfoList\":${SUBNET_JSON}}}"
        [[ "${HTTP_CODE}" == "200" ]] || { fail 0 "VPC create: ${BODY:0:200}"; return 1; }
    fi
    spider GET "/securitygroup/${SG_NAME}?ConnectionName=${CONNECTION_NAME}"
    if [[ "${HTTP_CODE}" != "200" ]]; then
        spider POST "/securitygroup" "{\"ConnectionName\":\"${CONNECTION_NAME}\",\"ReqInfo\":{\"Name\":\"${SG_NAME}\",
            \"VPCName\":\"${VPC_NAME}\",\"SecurityRules\":${SG_RULES_JSON}}}"
        [[ "${HTTP_CODE}" == "200" ]] || { fail 0 "SG create: ${BODY:0:200}"; return 1; }
    fi
    spider GET "/keypair/${KEYPAIR_NAME}?ConnectionName=${CONNECTION_NAME}"
    if [[ "${HTTP_CODE}" != "200" ]]; then
        spider POST "/keypair" "{\"ConnectionName\":\"${CONNECTION_NAME}\",\"ReqInfo\":{\"Name\":\"${KEYPAIR_NAME}\"}}"
        [[ "${HTTP_CODE}" == "200" ]] || { fail 0 "KeyPair create: ${BODY:0:200}"; return 1; }
    fi

    local subnets ng
    subnets=$(echo "${SUBNET_JSON}" | jq -c '[.[].Name]')
    ng="{\"Name\":\"${NODEGROUP_NAME}\",\"ImageName\":\"${NODE_IMAGE}\",\"VMSpecName\":\"${NODE_VM_SPEC}\",
         \"RootDiskType\":\"${NODE_ROOT_DISK_TYPE}\",\"RootDiskSize\":\"${NODE_ROOT_DISK_SIZE}\",
         \"KeyPairName\":\"${KEYPAIR_NAME}\",\"OnAutoScaling\":\"false\",
         \"DesiredNodeSize\":\"${NODE_SIZE}\",\"MinNodeSize\":\"${NODE_SIZE}\",\"MaxNodeSize\":\"${NODE_SIZE}\"}"

    spider GET "/cluster/${CLUSTER_NAME}?ConnectionName=${CONNECTION_NAME}"
    if [[ "${HTTP_CODE}" != "200" ]]; then
        local ng_list="[]"
        [[ "${CLUSTER_TYPE}" == "II" ]] && ng_list="[${ng}]"
        spider POST "/cluster" "{\"ConnectionName\":\"${CONNECTION_NAME}\",\"ReqInfo\":{\"Name\":\"${CLUSTER_NAME}\",
            \"Version\":\"${K8S_VERSION}\",\"VPCName\":\"${VPC_NAME}\",\"SubnetNames\":${subnets},
            \"SecurityGroupNames\":[\"${SG_NAME}\"],\"NodeGroupList\":${ng_list}}}"
        [[ "${HTTP_CODE}" == "200" ]] || { fail 0 "Cluster create: ${BODY:0:200}"; return 1; }
    fi
    wait_for "${CLUSTER_TIMEOUT}" 30 cluster_active \
        || { fail 0 "Cluster not Active within ${CLUSTER_TIMEOUT}s (last Status: ${CLUSTER_STATUS:-none})"; return 1; }

    if [[ "${CLUSTER_TYPE}" == "I" ]]; then
        spider GET "/cluster/${CLUSTER_NAME}?ConnectionName=${CONNECTION_NAME}"
        if [[ "$(echo "${BODY}" | jq '[.NodeGroupList[]?] | length')" == "0" ]]; then
            spider POST "/cluster/${CLUSTER_NAME}/nodegroup" "{\"ConnectionName\":\"${CONNECTION_NAME}\",\"ReqInfo\":${ng}}"
            [[ "${HTTP_CODE}" == "200" ]] || { fail 0 "NodeGroup add: ${BODY:0:200}"; return 1; }
        fi
    fi
    wait_for "${CLUSTER_TIMEOUT}" 30 nodegroups_active \
        || { fail 0 "NodeGroup not Active within ${CLUSTER_TIMEOUT}s (last Status: ${NODEGROUP_STATUS:-none})"; return 1; }

    echo "${BODY}" | jq -r '.AccessInfo.Kubeconfig // empty' > "${KUBECONFIG}"
    [[ -s "${KUBECONFIG}" ]] || { fail 0 "empty Kubeconfig"; return 1; }
    if grep -q '.spider-credential' "${KUBECONFIG}" && [[ ! -f "${SPIDER_CREDENTIAL}" ]]; then
        fail 0 "kubeconfig needs ${SPIDER_CREDENTIAL} (SPIDER_USERNAME, SPIDER_PASSWORD)"; return 1
    fi
    wait_for "${K8S_TIMEOUT}" 15 nodes_ready || { fail 0 "no Ready node (kubectl access?)"; return 1; }
    pass 0 "K8s $(echo "${BODY}" | jq -r '.Version')"
}

# ── Step 1: Driver Enable ────────────────────────────────────────────────────
step1() {
    driver_status
    note "before enable: $(echo "${DRIVER_JSON}" | jq -r '"\(.ManagementMode)/\(.Status)"')"
    spider PUT "/cluster/${CLUSTER_NAME}/storagedriver" "{\"ConnectionName\":\"${CONNECTION_NAME}\",
        \"ReqInfo\":{\"Kind\":\"Block\",\"Version\":\"${DRIVER_VERSION}\"}}"
    [[ "${HTTP_CODE}" == "200" ]] || { fail 1 "Enable: ${BODY:0:200}"; return 1; }
    FAILED_POLLS=0
    wait_for "${DRIVER_TIMEOUT}" 15 driver_enabled \
        || { fail 1 "Status ${DRIVER_STATUS:-none}, not Enabled within ${DRIVER_TIMEOUT}s"; return 1; }
    CSI_NAME=$(echo "${DRIVER_JSON}" | jq -r '.CSIDriverName // empty')
    MGMT_MODE=$(echo "${DRIVER_JSON}" | jq -r '.ManagementMode // empty')
    note "driver: $(echo "${DRIVER_JSON}" | jq -r '"\(.CSPDriverName) \(.Version // "") (\(.ManagementMode))"')"
    [[ -n "${CSI_NAME}" ]] || { fail 1 "CSIDriverName empty"; return 1; }
    pass 1
}

# ── Step 2: CSI Driver Installed ─────────────────────────────────────────────
step2() {
    wait_for "${K8S_TIMEOUT}" 15 csi_registered || { fail 2 "CSIDriver ${CSI_NAME} not registered on CSINode"; return 1; }
    pass 2 "${CSI_NAME}"
}

# ── Step 3: StorageClass ─────────────────────────────────────────────────────
step3() {
    local scs auto
    scs=$(kubectl get storageclass -o json 2>>"${LOG_FILE}" | jq -c --arg p "${CSI_NAME}" '[.items[] | select(.provisioner==$p)]')
    auto=$(echo "${scs}" | jq -r '[.[] | select((.reclaimPolicy // "Delete")=="Delete")]
        | (map(select(.metadata.annotations["storageclass.kubernetes.io/is-default-class"]=="true")) + .)[0].metadata.name // empty')
    if [[ -n "${auto}" ]]; then
        SC_NAME="${auto}"
        note "StorageClass: auto (${SC_NAME})"
    else
        SC_NAME="${TEST_SC_NAME}"
        kubectl apply -f - >>"${LOG_FILE}" 2>&1 <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata: { name: ${SC_NAME} }
provisioner: ${CSI_NAME}
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
parameters: ${SC_PARAMETERS_JSON}
EOF
        kubectl get storageclass "${SC_NAME}" >/dev/null 2>&1 || { fail 3 "StorageClass create failed"; return 1; }
        note "StorageClass: created (${SC_NAME}, no CSP-provided SC)"
    fi
    pass 3 "${SC_NAME}"
}

# ── Step 4: PVC -> PV -> Bound, CSP disk ─────────────────────────────────────
step4() {
    kubectl apply -f - >>"${LOG_FILE}" 2>&1 <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata: { name: ${PVC_NAME} }
spec:
  accessModes: ["ReadWriteOnce"]
  storageClassName: ${SC_NAME}
  resources: { requests: { storage: 10Gi } }
EOF
    apply_pod
    wait_for "${K8S_TIMEOUT}" 10 pvc_bound || { fail 4 "PVC not Bound: $(kubectl describe pvc "${PVC_NAME}" 2>&1 | tail -3 | tr '\n' ' ')"; return 1; }
    PV_NAME=$(kubectl get pvc "${PVC_NAME}" -o jsonpath='{.spec.volumeName}')
    VOLUME_HANDLE=$(kubectl get pv "${PV_NAME}" -o jsonpath='{.spec.csi.volumeHandle}')
    log "PV ${PV_NAME}, volumeHandle ${VOLUME_HANDLE}"
    [[ -n "${VOLUME_HANDLE}" ]] || { fail 4 "PV has no CSI volumeHandle"; return 1; }
    wait_for 300 15 csp_disk_exists || { fail 4 "CSP disk ${VOLUME_HANDLE} not found via Spider ListAllDisk"; return 1; }
    pass 4 "${VOLUME_HANDLE}"
}

# ── Step 5: Attach/Mount, R/W, re-attach ─────────────────────────────────────
step5() {
    local token read1 read2
    token="spider-$(date +%s)"
    wait_for "${K8S_TIMEOUT}" 10 pod_ready || { fail 5 "Pod not Ready: $(kubectl describe pod "${POD_NAME}" 2>&1 | tail -3 | tr '\n' ' ')"; return 1; }
    kubectl exec "${POD_NAME}" -- sh -c "echo ${token} > /data/test.txt && sync" >>"${LOG_FILE}" 2>&1
    read1=$(kubectl exec "${POD_NAME}" -- cat /data/test.txt 2>>"${LOG_FILE}")
    [[ "${read1}" == "${token}" ]] || { fail 5 "write/read mismatch"; return 1; }
    kubectl delete pod "${POD_NAME}" --wait=true --timeout=300s >>"${LOG_FILE}" 2>&1
    apply_pod
    wait_for "${K8S_TIMEOUT}" 10 pod_ready || { fail 5 "re-created Pod not Ready (re-attach)"; return 1; }
    read2=$(kubectl exec "${POD_NAME}" -- cat /data/test.txt 2>>"${LOG_FILE}")
    [[ "${read2}" == "${token}" ]] || { fail 5 "data lost after Pod re-create"; return 1; }
    pass 5
}

# ── Step 6: PVC Delete -> PV, CSP disk deleted ───────────────────────────────
step6() {
    delete_test_volume
    case $? in
        1) fail 6 "PV ${PV_NAME} not deleted"; return 1 ;;
        2) fail 6 "CSP disk ${VOLUME_HANDLE} still exists"; return 1 ;;
    esac
    pass 6
}

# ── Step 7: Driver Disable ───────────────────────────────────────────────────
step7() {
    local disabled=false
    spider DELETE "/cluster/${CLUSTER_NAME}/storagedriver/Block" "$(conn_body)"
    [[ "${HTTP_CODE}" == "200" && "$(echo "${BODY}" | jq -r '.Result // empty' 2>/dev/null)" == "true" ]] && disabled=true
    if [[ "${EXPECT_DISABLE}" == "reject" || "${MGMT_MODE}" == "BuiltIn" ]]; then
        if ! ${disabled}; then
            R[7]="➖"; note "S7: rejected as expected (${MGMT_MODE})"; return 1
        fi
        fail 7 "expected rejection, but Disable succeeded"; return 1
    fi
    ${disabled} || { fail 7 "Disable: HTTP ${HTTP_CODE} ${BODY:0:200}"; return 1; }
    wait_for "${DRIVER_TIMEOUT}" 15 driver_disabled \
        || { fail 7 "Status ${DRIVER_STATUS:-none}, not Disabled within ${DRIVER_TIMEOUT}s"; return 1; }
    # Status alone may not mean the driver is gone; step 8 proves a re-install only if it is
    wait_for "${K8S_TIMEOUT}" 15 csi_deregistered \
        || { fail 7 "Status Disabled, but ${CSI_NAME} still registered on CSINode"; return 1; }
    pass 7
}

# ── Step 8: Driver Re-Enable ─────────────────────────────────────────────────
step8() {
    spider PUT "/cluster/${CLUSTER_NAME}/storagedriver" "{\"ConnectionName\":\"${CONNECTION_NAME}\",\"ReqInfo\":{\"Kind\":\"Block\"}}"
    [[ "${HTTP_CODE}" == "200" ]] || { fail 8 "Re-Enable: ${BODY:0:200}"; return 1; }
    FAILED_POLLS=0
    wait_for "${DRIVER_TIMEOUT}" 15 driver_enabled \
        || { fail 8 "Status ${DRIVER_STATUS:-none}, not Enabled within ${DRIVER_TIMEOUT}s (no-op re-enable?)"; return 1; }
    wait_for "${K8S_TIMEOUT}" 15 csi_registered || { fail 8 "CSI driver not registered again"; return 1; }
    pass 8
}

# Step dependencies: 0 -> 1 -> 2 -> 3 -> 4 -> 5 -> 6, then 1 -> 7 -> 8
if step0 && step1; then
    step2 && step3 && step4 && step5 && step6
    # A failed step 4-6 can leave the PVC behind; release it while the driver is still installed
    if [[ "${R[6]}" != "✅" ]] && kubectl get pvc "${PVC_NAME}" >/dev/null 2>&1; then
        delete_test_volume || note "test volume not released before Disable (PV ${PV_NAME})"
    fi
    if step7; then
        step8
    elif [[ "${R[7]}" == "➖" ]]; then
        R[8]="➖"
    fi
fi

elapsed=$(( $(date +%s) - start_time ))
note "elapsed ${elapsed}s"
write_result    # visible while cleanup runs
trap - INT TERM # a signal now aborts cleanup
cleanup
write_result    # adds cleanup notes
log "Done. Log: ${LOG_FILE}"
