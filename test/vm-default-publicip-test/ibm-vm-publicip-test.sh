#!/bin/bash

# IBM VM Default-PublicIP Test Script
# Author: CB-Spider Team

export CSP_NAME="IBM"
export CONNECTION_NAME="ibm-us-east-1-config"
export VM_NAME="cb-spider-publicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_publicip_results}/result_ibm.txt"

export CREATE_JSON='{
  "ConnectionName": "ibm-us-east-1-config",
  "ReqInfo": {
    "Name": "cb-spider-publicip-test",
    "ImageName": "r014-1696a049-e959-493d-9a97-1655ef4c942e",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "bx2-2x8",
    "KeyPairName": "keypair-01"
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-vm-publicip-test.sh"
