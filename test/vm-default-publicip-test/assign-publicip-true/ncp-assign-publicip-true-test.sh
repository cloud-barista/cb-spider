#!/bin/bash

# NCP VM AssignPublicIP=true Test Script
# Author: CB-Spider Team

export CSP_NAME="NCP"
export CONNECTION_NAME="ncp-korea1-config"
export VM_NAME="cb-spider-truepublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_truepublicip_results}/result_ncp.txt"

export CREATE_JSON='{
  "ConnectionName": "ncp-korea1-config",
  "ReqInfo": {
    "Name": "cb-spider-truepublicip-test",
    "ImageName": "104630229",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "s2-g3",
    "KeyPairName": "keypair-01",
    "AssignPublicIP": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-true-test.sh"
