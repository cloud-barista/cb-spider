#!/bin/bash
#
# Create MyImage (VM snapshot) from VM-1
#   - waits until VM-1 is Running (max 40 x 15s = 10 min)
#   - API: POST /myimage {ConnectionName, ReqInfo:{Name, SourceVM}}   cf) spiderwatch
#     (the old 'spctl vm snapshot -d' command is no longer supported by spctl)
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
spider_req GET "myimage/${MYIMAGE_NAME}?ConnectionName=${CONN_CONFIG}"
if is_ok; then
	echo "[SKIP] MyImage already exists: ${MYIMAGE_NAME}"
	exit 0
fi

check_vm_running() {
	spider_req GET "vmstatus/${VM_NAME}?ConnectionName=${CONN_CONFIG}"
	local STATUS=$(echo "$RES_BODY" | jq -r '.Status // empty' 2>/dev/null)
	echo "  VM '${VM_NAME}' status: ${STATUS:-unknown}"
	[ "$STATUS" = "Running" ]
}
wait_until 40 15 "VM Running" check_vm_running || exit 1

echo "============== before create MyImage: '${MYIMAGE_NAME}' (source VM: ${VM_NAME})"
spider_req POST "myimage" '{
    "ConnectionName": "'"${CONN_CONFIG}"'",
    "ReqInfo": {
        "Name": "'"${MYIMAGE_NAME}"'",
        "SourceVM": "'"${VM_NAME}"'"
    }
}'
print_res "create MyImage: ${MYIMAGE_NAME}"
echo "============== after create MyImage: '${MYIMAGE_NAME}'"

echo -e "\n\n"
