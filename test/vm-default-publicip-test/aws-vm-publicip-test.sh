#!/bin/bash

# AWS VM Default-PublicIP Test Script
# Author: CB-Spider Team

export CSP_NAME="AWS"
export CONNECTION_NAME="aws-config01"
export VM_NAME="cb-spider-publicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_publicip_results}/result_aws.txt"

export CREATE_JSON='{
  "ConnectionName": "aws-config01",
  "ReqInfo": {
    "Name": "cb-spider-publicip-test",
    "ImageName": "ami-0131a0fdbb6fda7e6",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "t2.micro",
    "KeyPairName": "keypair-01"
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-vm-publicip-test.sh"
