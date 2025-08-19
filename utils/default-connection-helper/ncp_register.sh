#!/bin/bash

# Source common functions
source ./common_register.sh

# Load NCP VPC credentials
NCP_CREDENTIAL_FILE="$HOME/.cb-spider/.ncp-credential"
check_credential_file "$NCP_CREDENTIAL_FILE" "NCP VPC credential file not found"

# Check required variables
check_required_vars "ncp_client_id" "ncp_client_secret"

# Register NCP VPC driver
register_driver "ncp-driver" "NCP" "ncp-driver-v1.0.so"

# Register NCP VPC credential
NCP_KEY_VALUES='[
  {"Key":"ClientId", "Value":"'"$ncp_client_id"'"},
  {"Key":"ClientSecret", "Value":"'"$ncp_client_secret"'"}
]'
register_credential "ncp-credential" "NCP" "$NCP_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "ncp-driver" "ncp-credential" "NCP" "ncp"
