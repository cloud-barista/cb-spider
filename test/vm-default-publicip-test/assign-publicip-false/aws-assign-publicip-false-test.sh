#!/bin/bash

# AWS VM AssignPublicIP=false Test Script
# Author: CB-Spider Team

export CSP_NAME="AWS"
export CONNECTION_NAME="aws-config01"
export VM_NAME="cb-spider-nopublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_nopublicip_results}/result_aws.txt"

export CREATE_JSON='{
  "ConnectionName": "aws-config01",
  "ReqInfo": {
    "Name": "cb-spider-nopublicip-test",
    "ImageName": "ami-0131a0fdbb6fda7e6",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "t2.micro",
    "KeyPairName": "keypair-01",
    "AssignPublicIP": false
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-false-test.sh"
