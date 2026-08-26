#!/bin/bash

# GCP VM Default-PublicIP Test Script
# Author: CB-Spider Team

export CSP_NAME="GCP"
export CONNECTION_NAME="gcp-iowa-config"
export VM_NAME="cb-spider-publicip-test"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_publicip_results}/result_gcp.txt"

export CREATE_JSON='{
  "ConnectionName": "gcp-iowa-config",
  "ReqInfo": {
    "Name": "cb-spider-publicip-test",
    "ImageName": "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-2404-noble-amd64-v20240423",
    "VPCName": "vpc-01",
    "SubnetName": "subnet-01",
    "SecurityGroupNames": ["sg-01"],
    "VMSpecName": "e2-standard-2",
    "KeyPairName": "keypair-01"
  }
}'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-vm-publicip-test.sh"
