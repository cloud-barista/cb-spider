#!/bin/bash

# Source common functions
source ./common_register.sh

# Load NCP Classic credentials
NCP_CREDENTIAL_FILE="./.ncpclassic-credential"
check_credential_file "$NCP_CREDENTIAL_FILE" "NCP Classic credential file not found"

# Check required variables
check_required_vars "ncpclassic_client_id" "ncpclassic_client_secret"

# Register NCP Classic driver
register_driver "ncpclassic-driver" "NCP" "ncp-driver-v1.0.so"

# Register NCP Classic credential
NCP_KEY_VALUES='[
  {"Key":"ClientId", "Value":"'"$ncpclassic_client_id"'"},
  {"Key":"ClientSecret", "Value":"'"$ncpclassic_client_secret"'"}
]'
register_credential "ncpclassic-credential" "NCP" "$NCP_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "ncpclassic-driver" "ncpclassic-credential" "NCP" "ncpclassic"
