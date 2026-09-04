#!/bin/bash

# OpenStack VM AssignPublicIP=true Test Script
# Author: CB-Spider Team

export CSP_NAME="OPENSTACK"
export CONNECTION_NAME="openstack-config01"
export VM_NAME="cb-spider-truepublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_truepublicip_results}/result_openstack.txt"

export CREATE_JSON='{
  "ConnectionName": "openstack-config01",
  "ReqInfo": {
    "Name": "cb-spider-truepublicip-test",
    "ImageName": "78d90dae-d21d-4606-a9dd-c1268e321864",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "m1.small",
    "KeyPairName": "keypair-01",
    "AssignPublicIP": true
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-true-test.sh"
