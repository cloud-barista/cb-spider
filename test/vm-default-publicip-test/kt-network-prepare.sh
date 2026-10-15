#!/bin/bash

# KT Cloud Basic Resource Prepare Script (VPC/Subnet -> SecurityGroup -> KeyPair)
# Author: CB-Spider Team

export CSP_NAME="KT"
export CONNECTION_NAME="kt-mokdong1-config"
export VPC_NAME="vpc-01"
export VPC_CIDR="10.0.0.0/16"
export SUBNET_JSON='[{"Name": "subnet-01", "IPv4_CIDR": "10.29.102.0/24"}]'
export SG_NAME="sg-01"
export KEYPAIR_NAME="keypair-01"
export RESULT_FILE="${RESULT_DIR:-/tmp/vm_publicip_network_results}/result_kt.txt"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-network-prepare.sh"
