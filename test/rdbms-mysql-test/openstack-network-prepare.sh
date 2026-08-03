#!/bin/bash

# OpenStack Network Prerequisite Prepare Script
# Creates vpc-01/subnet-01, as used by openstack-rdbms-test.sh.
# Author: CB-Spider Team

export CSP_NAME="OPENSTACK"
export CONNECTION_NAME="openstack-config01"
export VPC_NAME="vpc-01"
export VPC_CIDR="10.0.0.0/16"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_network_results}/result_openstack.txt"

export SUBNET_JSON='[{"Name": "subnet-01", "IPv4_CIDR": "10.0.1.0/24"}]'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-network-prepare.sh"
