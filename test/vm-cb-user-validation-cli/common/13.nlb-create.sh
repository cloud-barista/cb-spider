#!/bin/bash
#
# Create NLB (deployment only; load-balancing behavior is not tested)
#   - cf) spiderwatch: NLB is created without VMs, except for CSPs that
#         require a VM at creation time (NLB_VM_REQUIRED_AT_CREATE=true, e.g. KT Cloud)
#

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn|kt'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)   # this common/ directory (independent of CBSPIDER_ROOT)
source $SETUP_PATH/setup.env $1
source $SETUP_PATH/rest_func.sh

VM_NAME=${VM_NAME}-1

# skip if it already exists
spider_req GET "nlb/${NLB_NAME}?ConnectionName=${CONN_CONFIG}"
if is_ok; then
	echo "[SKIP] NLB already exists: ${NLB_NAME}"
	exit 0
fi

VMS="[]"
if [ "$NLB_VM_REQUIRED_AT_CREATE" = "true" ]; then
	check_vm_running() {
		spider_req GET "vmstatus/${VM_NAME}?ConnectionName=${CONN_CONFIG}"
		local STATUS=$(echo "$RES_BODY" | jq -r '.Status // empty' 2>/dev/null)
		echo "  VM '${VM_NAME}' status: ${STATUS:-unknown}"
		[ "$STATUS" = "Running" ]
	}
	wait_until 40 15 "VM Running" check_vm_running || exit 1
	VMS="[\"${VM_NAME}\"]"
fi

echo "============== before create NLB: '${NLB_NAME}'"
spider_req POST "nlb" '{
    "ConnectionName": "'"${CONN_CONFIG}"'",
    "ReqInfo": {
        "Name": "'"${NLB_NAME}"'",
        "VPCName": "'"${VPC_NAME}"'",
        "Type": "'"${NLB_TYPE}"'",
        "Scope": "'"${NLB_SCOPE}"'",
        "Listener": {
            "Protocol": "'"${NLB_LISTENER_PROTOCOL}"'",
            "Port": "'"${NLB_LISTENER_PORT}"'"
        },
        "VMGroup": {
            "Protocol": "'"${NLB_TARGET_PROTOCOL}"'",
            "Port": "'"${NLB_TARGET_PORT}"'",
            "VMs": '"${VMS}"'
        },
        "HealthChecker": {
            "Protocol": "'"${NLB_HEALTH_PROTOCOL}"'",
            "Port": "'"${NLB_HEALTH_PORT}"'",
            "Interval": "default",
            "Timeout": "default",
            "Threshold": "default"
        }
    }
}'
print_res "create NLB: ${NLB_NAME}"
echo "============== after create NLB: '${NLB_NAME}'"

echo -e "\n\n"
