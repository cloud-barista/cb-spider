#!/bin/bash

# Source common functions
source ./common_register.sh

# Load NCP VPC credentials
NCP_CREDENTIAL_FILE="$HOME/.cb-spider/.ncpvpc-credential"
check_credential_file "$NCP_CREDENTIAL_FILE" "NCP VPC credential file not found"

# Check required variables
check_required_vars "ncpvpc_client_id" "ncpvpc_client_secret"

# Register NCP VPC driver
register_driver "ncpvpc-driver" "NCPVPC" "ncp-driver-v1.0.so"

# Register NCP VPC credential
NCP_KEY_VALUES='[
  {"Key":"ClientId", "Value":"'"$ncpvpc_client_id"'"},
  {"Key":"ClientSecret", "Value":"'"$ncpvpc_client_secret"'"}
]'
register_credential "ncpvpc-credential" "NCPVPC" "$NCP_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "ncpvpc-driver" "ncpvpc-credential" "NCPVPC" "ncpvpc"
