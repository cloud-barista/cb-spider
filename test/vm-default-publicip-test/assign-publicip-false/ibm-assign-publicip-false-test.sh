#!/bin/bash

# IBM VM AssignPublicIP=false Test Script
# Author: CB-Spider Team

export CSP_NAME="IBM"
export CONNECTION_NAME="ibm-us-east-1-config"
export VM_NAME="cb-spider-nopublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_nopublicip_results}/result_ibm.txt"

export CREATE_JSON='{
  "ConnectionName": "ibm-us-east-1-config",
  "ReqInfo": {
    "Name": "cb-spider-nopublicip-test",
    "ImageName": "r014-1696a049-e959-493d-9a97-1655ef4c942e",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "bx2-2x8",
    "KeyPairName": "keypair-01",
    "AssignPublicIP": false
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-false-test.sh"
