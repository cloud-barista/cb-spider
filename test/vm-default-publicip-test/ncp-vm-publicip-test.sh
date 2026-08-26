#!/bin/bash

# NCP VM Default-PublicIP Test Script
# Author: CB-Spider Team

export CSP_NAME="NCP"
export CONNECTION_NAME="ncp-korea1-config"
export VM_NAME="cb-spider-publicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_publicip_results}/result_ncp.txt"

export CREATE_JSON='{
  "ConnectionName": "ncp-korea1-config",
  "ReqInfo": {
    "Name": "cb-spider-publicip-test",
    "ImageName": "104630229",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "s2-g3",
    "KeyPairName": "keypair-01"
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-vm-publicip-test.sh"
