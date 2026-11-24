#!/bin/bash

# NHN Cloud VM Default-PublicIP Test Script
# Author: CB-Spider Team

export CSP_NAME="NHN"
export CONNECTION_NAME="nhn-korea-pangyo1-config"
export VM_NAME="cb-spider-publicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_publicip_results}/result_nhn.txt"

export CREATE_JSON='{
  "ConnectionName": "nhn-korea-pangyo1-config",
  "ReqInfo": {
    "Name": "cb-spider-publicip-test",
    "ImageName": "1cc42a4d-53da-4da9-a464-5241839b68c5",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "m2.c4m8",
    "KeyPairName": "keypair-01"
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-vm-publicip-test.sh"
