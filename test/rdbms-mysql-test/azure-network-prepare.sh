#!/bin/bash

# Azure Network Prerequisite Prepare Script
# Creates vpc-01/subnet-01, as used by azure-rdbms-test.sh.
# Author: CB-Spider Team

export CSP_NAME="AZURE"
export CONNECTION_NAME="azure-koreacentral-config"
export VPC_NAME="vpc-01"
export VPC_CIDR="10.0.0.0/16"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_network_results}/result_azure.txt"

export SUBNET_JSON='[{"Name": "subnet-01", "IPv4_CIDR": "10.0.1.0/24"}]'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-network-prepare.sh"
