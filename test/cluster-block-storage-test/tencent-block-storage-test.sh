#!/bin/bash

# Tencent Cluster Block Storage Driver Test
# Type-I. Set NODE_VM_SPEC and NODE_IMAGE (OS name from the TKE list).
# Override any value below with an env var of the same name.
# Author: CB-Spider Team

export CSP_NAME="Tencent"
export CSP_KEY="tencent"
export CONNECTION_NAME="${CONNECTION_NAME:-tencent-beijing3-config}"
export CLUSTER_TYPE="I"                       # I: NodeGroup added after Cluster, II: NodeGroup with Cluster
export VPC_CIDR="10.0.0.0/16"
export NODE_VM_SPEC="${NODE_VM_SPEC:-SET_ME}"
export NODE_IMAGE="${NODE_IMAGE:-SET_ME}"            # empty: CSP default node image
export EXPECT_DISABLE="${EXPECT_DISABLE:-ok}"    # ok | reject

# Not using "${SUBNET_JSON:-...}": bash ends the expansion at the first unescaped '}'.
if [[ -z "${SUBNET_JSON}" ]]; then
    export SUBNET_JSON="[{\"Name\":\"subnet-01\",\"IPv4_CIDR\":\"10.0.1.0/24\",\"Zone\":\"ap-beijing-3\"}]"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${SCRIPT_DIR}/common-block-storage-test.sh"
