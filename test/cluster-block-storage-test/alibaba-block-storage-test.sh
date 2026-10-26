#!/bin/bash

# Alibaba Cluster Block Storage Driver Test
# Type-I. Disable is rejected by design (csi-plugin/csi-provisioner are shared with File).
# Override any value below with an env var of the same name.
# Author: CB-Spider Team

export CSP_NAME="Alibaba"
export CSP_KEY="alibaba"
export CONNECTION_NAME="${CONNECTION_NAME:-alibaba-beijing-config}"
export CLUSTER_TYPE="I"                       # I: NodeGroup added after Cluster, II: NodeGroup with Cluster
export VPC_CIDR="10.0.0.0/16"
export NODE_VM_SPEC="${NODE_VM_SPEC:-ecs.g6.large}"
export NODE_IMAGE="${NODE_IMAGE:-}"            # empty: CSP default node image
export EXPECT_DISABLE="${EXPECT_DISABLE:-reject}"    # ok | reject

# Not using "${SUBNET_JSON:-...}": bash ends the expansion at the first unescaped '}'.
if [[ -z "${SUBNET_JSON}" ]]; then
    export SUBNET_JSON="[{\"Name\":\"subnet-01\",\"IPv4_CIDR\":\"10.0.1.0/24\",\"Zone\":\"cn-beijing-f\"}]"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${SCRIPT_DIR}/common-block-storage-test.sh"
