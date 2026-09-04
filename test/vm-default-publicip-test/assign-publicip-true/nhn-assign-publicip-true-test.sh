#!/bin/bash

# NHN Cloud VM AssignPublicIP=true Test Script
# Author: CB-Spider Team

export CSP_NAME="NHN"
export CONNECTION_NAME="nhn-korea-pangyo1-config"
export VM_NAME="cb-spider-truepublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_truepublicip_results}/result_nhn.txt"

export CREATE_JSON='{
  "ConnectionName": "nhn-korea-pangyo1-config",
  "ReqInfo": {
    "Name": "cb-spider-truepublicip-test",
    "ImageName": "5396655e-166a-4875-80d2-ed8613aa054f",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "m2.c4m8",
    "KeyPairName": "keypair-01",
    "AssignPublicIP": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-true-test.sh"
