#!/bin/bash

# KT Cloud VM AssignPublicIP=false Test Script
# Note: KT Cloud VPC enforces Security Groups only via per-PublicIP
# PortForwarding/Firewall rules, so when AssignPublicIP=false the requested
# SecurityGroups are NOT enforced at the network level for this VM (see
# cloud-control-manager/cloud-driver/drivers/kt/resources/VMHandler.go).
# The VM itself must still reach Running with a PrivateIP and no PublicIP.
# Author: CB-Spider Team

export CSP_NAME="KT"
export CONNECTION_NAME="kt-mokdong1-config"
export VM_NAME="cb-spider-nopublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_nopublicip_results}/result_kt.txt"

export CREATE_JSON='{
  "ConnectionName": "kt-mokdong1-config",
  "ReqInfo": {
    "Name": "cb-spider-nopublicip-test",
    "ImageName": "1a772df6-262e-43a7-896f-98fa23d715c7",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "4x8.itl",
    "KeyPairName": "keypair-01",
    "AssignPublicIP": false
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-false-test.sh"
