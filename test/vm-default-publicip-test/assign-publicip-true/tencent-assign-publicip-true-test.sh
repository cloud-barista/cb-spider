#!/bin/bash

# Tencent VM AssignPublicIP=true Test Script
# Author: CB-Spider Team

export CSP_NAME="TENCENT"
export CONNECTION_NAME="tencent-beijing3-config"
export VM_NAME="cb-spider-truepublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_truepublicip_results}/result_tencent.txt"

export CREATE_JSON='{
  "ConnectionName": "tencent-beijing3-config",
  "ReqInfo": {
    "Name": "cb-spider-truepublicip-test",
    "ImageName": "img-pi0ii46r",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "S5.MEDIUM8",
    "KeyPairName": "keypair-01",
    "AssignPublicIP": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-true-test.sh"
