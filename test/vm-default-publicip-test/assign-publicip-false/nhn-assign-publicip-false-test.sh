#!/bin/bash

# NHN Cloud VM AssignPublicIP=false Test Script
# Author: CB-Spider Team

export CSP_NAME="NHN"
export CONNECTION_NAME="nhn-korea-pangyo1-config"
export VM_NAME="cb-spider-nopublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_nopublicip_results}/result_nhn.txt"

export CREATE_JSON='{
  "ConnectionName": "nhn-korea-pangyo1-config",
  "ReqInfo": {
    "Name": "cb-spider-nopublicip-test",
    "ImageName": "1cc42a4d-53da-4da9-a464-5241839b68c5",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "m2.c4m8",
    "KeyPairName": "keypair-01",
    "AssignPublicIP": false
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-false-test.sh"
