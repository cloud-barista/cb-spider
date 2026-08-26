#!/bin/bash

# KT Cloud VM Default-PublicIP Test Script
# Author: CB-Spider Team

export CSP_NAME="KT"
export CONNECTION_NAME="kt-mokdong1-config"
export VM_NAME="cb-spider-publicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_publicip_results}/result_kt.txt"

export CREATE_JSON='{
  "ConnectionName": "kt-mokdong1-config",
  "ReqInfo": {
    "Name": "cb-spider-publicip-test",
    "ImageName": "1a772df6-262e-43a7-896f-98fa23d715c7",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "4x8.itl",
    "KeyPairName": "keypair-01"
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-vm-publicip-test.sh"
