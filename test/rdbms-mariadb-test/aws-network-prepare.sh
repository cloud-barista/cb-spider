#!/bin/bash

# AWS Network Prerequisite Prepare Script
# Creates vpc-01 (10.0.0.0/16) with subnet-01/subnet-02 in two different AZs
# (required for the RDS SubnetGroup) and sg-01, as used by aws-rdbms-test.sh.
# Override AWS_AZ1/AWS_AZ2 to match your account's available AZs.
# Author: CB-Spider Team

export CSP_NAME="AWS"
export CONNECTION_NAME="aws-config01"
export VPC_NAME="vpc-01"
export VPC_CIDR="10.0.0.0/16"
export RESULT_FILE="${RESULT_DIR:-/tmp/rdbms_network_results}/result_aws.txt"

AWS_AZ1="${AWS_AZ1:-ap-southeast-2a}"
AWS_AZ2="${AWS_AZ2:-ap-southeast-2b}"

export SUBNET_JSON="[
  {\"Name\": \"subnet-01\", \"IPv4_CIDR\": \"10.0.1.0/24\", \"Zone\": \"${AWS_AZ1}\"},
  {\"Name\": \"subnet-02\", \"IPv4_CIDR\": \"10.0.2.0/24\", \"Zone\": \"${AWS_AZ2}\"}
]"

export SG_NAME="sg-01"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-network-prepare.sh"
