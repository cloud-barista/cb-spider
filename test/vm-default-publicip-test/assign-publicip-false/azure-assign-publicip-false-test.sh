#!/bin/bash

# Azure VM AssignPublicIP=false Test Script
# Author: CB-Spider Team

export CSP_NAME="AZURE"
export CONNECTION_NAME="azure-koreacentral-config"
export VM_NAME="cb-spider-nopublicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_nopublicip_results}/result_azure.txt"

export CREATE_JSON='{
  "ConnectionName": "azure-koreacentral-config",
  "ReqInfo": {
    "Name": "cb-spider-nopublicip-test",
    "ImageName": "Canonical:ubuntu-25_04-daily:minimal:25.04.202601140",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "Standard_B1ls",
    "KeyPairName": "keypair-01",
    "AssignPublicIP": false
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-assign-publicip-false-test.sh"
