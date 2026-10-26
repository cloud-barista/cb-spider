#!/bin/bash

# AWS Cluster Block Storage Driver Test
# Type-I. EKS needs 2+ subnets in different AZs (override AWS_AZ1/AWS_AZ2).
# Override any value below with an env var of the same name.
# Author: CB-Spider Team

export CSP_NAME="AWS"
export CSP_KEY="aws"
export CONNECTION_NAME="${CONNECTION_NAME:-aws-config01}"
export CLUSTER_TYPE="I"                       # I: NodeGroup added after Cluster, II: NodeGroup with Cluster
export VPC_CIDR="10.0.0.0/16"
export NODE_VM_SPEC="${NODE_VM_SPEC:-t3.medium}"
export NODE_IMAGE="${NODE_IMAGE:-}"            # empty: CSP default node image
export EXPECT_DISABLE="${EXPECT_DISABLE:-ok}"    # ok | reject

# Not using "${SUBNET_JSON:-...}": bash ends the expansion at the first unescaped '}'.
if [[ -z "${SUBNET_JSON}" ]]; then
    export SUBNET_JSON="[{\"Name\":\"subnet-01\",\"IPv4_CIDR\":\"10.0.1.0/24\",\"Zone\":\"${AWS_AZ1:-ap-southeast-2a}\"},
                   {\"Name\":\"subnet-02\",\"IPv4_CIDR\":\"10.0.2.0/24\",\"Zone\":\"${AWS_AZ2:-ap-southeast-2b}\"}]"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${SCRIPT_DIR}/common-block-storage-test.sh"
