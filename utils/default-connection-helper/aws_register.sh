#!/bin/bash

# Source common functions
source ./common_register.sh

# Load AWS credentials
AWS_CREDENTIAL_FILE="./.aws-credential"
check_credential_file "$AWS_CREDENTIAL_FILE" "AWS credential file not found"

# Check required variables
check_required_vars "aws_access_key_id" "aws_secret_access_key"

# Register AWS driver
register_driver "aws-driver" "AWS" "aws-driver-v1.0.so"

# Register AWS credential
AWS_KEY_VALUES='[
  {"Key": "aws_access_key_id", "Value": "'"$aws_access_key_id"'"},
  {"Key": "aws_secret_access_key", "Value": "'"$aws_secret_access_key"'"}
]'
register_credential "aws-credential" "AWS" "$AWS_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "aws-driver" "aws-credential" "AWS" "aws"
