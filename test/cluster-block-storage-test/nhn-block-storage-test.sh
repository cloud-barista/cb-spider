#!/bin/bash

# NHN Cluster Block Storage Driver Test
# Type-II. Set NODE_VM_SPEC. NKS provides no StorageClass, so step 3 creates one.
# Override any value below with an env var of the same name.
# Author: CB-Spider Team

export CSP_NAME="NHN"
export CSP_KEY="nhn"
export CONNECTION_NAME="${CONNECTION_NAME:-nhn-korea-pangyo1-config}"
export CLUSTER_TYPE="II"                       # I: NodeGroup added after Cluster, II: NodeGroup with Cluster
export VPC_CIDR="10.0.0.0/16"
export NODE_VM_SPEC="${NODE_VM_SPEC:-SET_ME}"
export NODE_IMAGE="${NODE_IMAGE:-}"            # empty: CSP default node image
export EXPECT_DISABLE="${EXPECT_DISABLE:-ok}"    # ok | reject

# Not using "${SUBNET_JSON:-...}": bash ends the expansion at the first unescaped '}'.
if [[ -z "${SUBNET_JSON}" ]]; then
    export SUBNET_JSON="[{\"Name\":\"subnet-01\",\"IPv4_CIDR\":\"10.0.1.0/24\",\"Zone\":\"kr-pub-a\"}]"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${SCRIPT_DIR}/common-block-storage-test.sh"
