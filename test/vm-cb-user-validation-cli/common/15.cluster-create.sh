#!/bin/bash
#
# Create Cluster and NodeGroup (deployment only; Pod deployment is not tested)
#   cf) spiderwatch testClusterCRUD
#   - type1 (AWS, Alibaba, Tencent): create Cluster => wait Active => add NodeGroup
#   - type2 (Azure, GCP, IBM, NCP, NHN): create Cluster with NodeGroup
#   - then wait until NodeGroup is Active
#   - wait: 120 x 30s = 1 hour each
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

if [ "$CLUSTER_SUPPORT" != "true" ]; then
	echo "[SKIP] Cluster is not supported for $1"
	exit 0
fi

MAX_ATTEMPTS=120
POLL_INTERVAL=30

NG_INFO='{
    "Name": "'"${NODEGROUP_NAME}"'",
    "ImageName": "'"${NODEGROUP_IMAGE_NAME}"'",
    "VMSpecName": "'"${NODEGROUP_SPEC_NAME}"'",
    "RootDiskType": "'"${NODEGROUP_ROOT_DISK_TYPE}"'",
    "RootDiskSize": "'"${NODEGROUP_ROOT_DISK_SIZE}"'",
    "KeyPairName": "'"${KEYPAIR_NAME}"'",
    "OnAutoScaling": "'"${NODEGROUP_ON_AUTO_SCALING}"'",
    "DesiredNodeSize": "'"${NODEGROUP_DESIRED_NODE_SIZE}"'",
    "MinNodeSize": "'"${NODEGROUP_MIN_NODE_SIZE}"'",
    "MaxNodeSize": "'"${NODEGROUP_MAX_NODE_SIZE}"'"
}'

#--- 1. extra subnet (e.g. AWS EKS requires subnets in 2+ AZs)
if [ -n "$EXTRA_SUBNET_CIDR" ]; then
	echo "============== before add extra Subnet: '${EXTRA_SUBNET_NAME}' (${EXTRA_SUBNET_CIDR}, ${EXTRA_SUBNET_ZONE})"
	spider_req POST "vpc/${VPC_NAME}/subnet" '{
	    "ConnectionName": "'"${CONN_CONFIG}"'",
	    "ReqInfo": {
	        "Name": "'"${EXTRA_SUBNET_NAME}"'",
	        "Zone": "'"${EXTRA_SUBNET_ZONE}"'",
	        "IPv4_CIDR": "'"${EXTRA_SUBNET_CIDR}"'"
	    }
	}'
	if is_ok || echo "$RES_BODY" | grep -qi 'already exist'; then
		echo "[OK]   extra Subnet: ${EXTRA_SUBNET_NAME}"
	else
		print_res "add extra Subnet: ${EXTRA_SUBNET_NAME}"
		exit 1
	fi
fi

#--- 2. subnet names registered in the VPC
spider_req GET "vpc/${VPC_NAME}?ConnectionName=${CONN_CONFIG}"
if ! is_ok; then
	print_res "get VPC: ${VPC_NAME}"
	exit 1
fi
SUBNET_NAMES=$(echo "$RES_BODY" | jq -c '[.SubnetInfoList[]?.IId.NameId | select(. != "")]')
if [ "$SUBNET_NAMES" = "[]" ] || [ -z "$SUBNET_NAMES" ]; then
	echo "[FAIL] VPC '${VPC_NAME}' has no subnets"
	exit 1
fi

if [ "$NODEGROUP_TYPE" = "type2" ]; then
	NG_LIST="[${NG_INFO}]"
else
	NG_LIST="[]"
fi

#--- 3. create Cluster (skip if it already exists)
spider_req GET "cluster/${CLUSTER_NAME}?ConnectionName=${CONN_CONFIG}"
if is_ok; then
	echo "[SKIP] Cluster already exists: ${CLUSTER_NAME}"
else
	echo "============== before create Cluster: '${CLUSTER_NAME}' (version: ${CLUSTER_VERSION:-default}, nodegroup: ${NODEGROUP_TYPE})"
	spider_req POST "cluster" '{
	    "ConnectionName": "'"${CONN_CONFIG}"'",
	    "ReqInfo": {
	        "Name": "'"${CLUSTER_NAME}"'",
	        "Version": "'"${CLUSTER_VERSION}"'",
	        "VPCName": "'"${VPC_NAME}"'",
	        "SubnetNames": '"${SUBNET_NAMES}"',
	        "SecurityGroupNames": ["'"${SG_NAME}"'"],
	        "NodeGroupList": '"${NG_LIST}"'
	    }
	}'
	print_res "create Cluster: ${CLUSTER_NAME}"
	is_ok || exit 1
fi

#--- 4. wait Cluster Active
check_cluster_active() {
	spider_req GET "cluster/${CLUSTER_NAME}?ConnectionName=${CONN_CONFIG}"
	is_ok || return 1   # transient error: keep polling
	local STATUS=$(echo "$RES_BODY" | jq -r '.Status // empty')
	echo "  Cluster '${CLUSTER_NAME}' status: ${STATUS}"
	[ "$STATUS" = "Active" ] && return 0
	return 1
}
wait_until $MAX_ATTEMPTS $POLL_INTERVAL "Cluster Active" check_cluster_active || exit 1
echo "[OK]   Cluster Active: ${CLUSTER_NAME}"

#--- 5. type1: add NodeGroup after Cluster is Active (skip if it already exists)
if [ "$NODEGROUP_TYPE" != "type2" ]; then
	HAS_NG=$(echo "$RES_BODY" | jq -r --arg n "$NODEGROUP_NAME" '[.NodeGroupList[]? | select(.IId.NameId == $n)] | length')
	if [ "$HAS_NG" != "0" ]; then
		echo "[SKIP] NodeGroup already exists: ${NODEGROUP_NAME}"
	else
		echo "============== before add NodeGroup: '${NODEGROUP_NAME}'"
		spider_req POST "cluster/${CLUSTER_NAME}/nodegroup" '{
		    "ConnectionName": "'"${CONN_CONFIG}"'",
		    "ReqInfo": '"${NG_INFO}"'
		}'
		print_res "add NodeGroup: ${NODEGROUP_NAME}"
		is_ok || exit 1
	fi
fi

#--- 6. wait NodeGroup Active
check_ng_active() {
	spider_req GET "cluster/${CLUSTER_NAME}?ConnectionName=${CONN_CONFIG}"
	is_ok || return 1
	local STATUS=$(echo "$RES_BODY" | jq -r --arg n "$NODEGROUP_NAME" \
		'[.NodeGroupList[]? | select(.IId.NameId == $n) | .Status] | first // "NotFound"')
	echo "  NodeGroup '${NODEGROUP_NAME}' status: ${STATUS}"
	[ "$STATUS" = "Active" ] && return 0
	return 1
}
wait_until $MAX_ATTEMPTS $POLL_INTERVAL "NodeGroup Active" check_ng_active || exit 1
echo "[OK]   NodeGroup Active: ${NODEGROUP_NAME}"

spider_req GET "cluster/${CLUSTER_NAME}?ConnectionName=${CONN_CONFIG}"
echo "$RES_BODY" | jq '{Name: .IId.NameId, Version, Status, NodeGroups: [.NodeGroupList[]? | {Name: .IId.NameId, Status, Nodes: [.Nodes[]?.NameId]}]}' 2>/dev/null
echo "============== after create Cluster: '${CLUSTER_NAME}'"

echo -e "\n\n"
