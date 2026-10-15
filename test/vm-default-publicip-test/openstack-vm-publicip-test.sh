#!/bin/bash

# OpenStack VM Default-PublicIP Test Script
# Author: CB-Spider Team

export CSP_NAME="OPENSTACK"
export CONNECTION_NAME="openstack-config01"
export VM_NAME="cb-spider-publicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_publicip_results}/result_openstack.txt"

export CREATE_JSON='{
  "ConnectionName": "openstack-config01",
  "ReqInfo": {
    "Name": "cb-spider-publicip-test",
    "ImageName": "78d90dae-d21d-4606-a9dd-c1268e321864",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "m1.small",
    "KeyPairName": "keypair-01"
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-vm-publicip-test.sh"
