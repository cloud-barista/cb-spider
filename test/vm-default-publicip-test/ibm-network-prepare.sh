#!/bin/bash

# IBM Basic Resource Prepare Script (VPC/Subnet -> SecurityGroup -> KeyPair)
# Author: CB-Spider Team

export CSP_NAME="IBM"
export CONNECTION_NAME="ibm-us-east-1-config"
export VPC_NAME="vpc-01"
export VPC_CIDR="192.168.0.0/16"
export SUBNET_JSON='[{"Name": "subnet-01", "IPv4_CIDR": "192.168.1.0/24"}]'
export SG_NAME="sg-01"
export KEYPAIR_NAME="keypair-01"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_publicip_network_results}/result_ibm.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-network-prepare.sh"
