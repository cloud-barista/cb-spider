#!/bin/bash

# GCP Cluster Block Storage Driver Test
# Type-II. GKE Standard cluster (Autopilot reports BuiltIn: Disable rejection is expected automatically).
# Override any value below with an env var of the same name.
# Author: CB-Spider Team

export CSP_NAME="GCP"
export CSP_KEY="gcp"
export CONNECTION_NAME="${CONNECTION_NAME:-gcp-iowa-config}"
export CLUSTER_TYPE="II"                       # I: NodeGroup added after Cluster, II: NodeGroup with Cluster
export VPC_CIDR="10.0.0.0/16"
export NODE_VM_SPEC="${NODE_VM_SPEC:-e2-standard-2}"
export NODE_IMAGE="${NODE_IMAGE:-}"            # empty: CSP default node image
export EXPECT_DISABLE="${EXPECT_DISABLE:-ok}"    # ok | reject

# Not using "${SUBNET_JSON:-...}": bash ends the expansion at the first unescaped '}'.
if [[ -z "${SUBNET_JSON}" ]]; then
    export SUBNET_JSON="[{\"Name\":\"subnet-01\",\"IPv4_CIDR\":\"10.0.1.0/24\"}]"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${SCRIPT_DIR}/common-block-storage-test.sh"
