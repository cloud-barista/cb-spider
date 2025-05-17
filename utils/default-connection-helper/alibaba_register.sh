#!/bin/bash

# Source common functions
source ./common_register.sh

# Load Alibaba credentials
ALIBABA_CREDENTIAL_FILE="./.alibaba-credential"
check_credential_file "$ALIBABA_CREDENTIAL_FILE" "Alibaba credential file not found"

# Check required variables
check_required_vars "alibaba_client_id" "alibaba_client_secret"

# Register Alibaba driver
register_driver "alibaba-driver" "ALIBABA" "alibaba-driver-v1.0.so"

# Register Alibaba credential
ALIBABA_KEY_VALUES='[
  {"Key":"ClientId", "Value":"'"$alibaba_client_id"'"},
  {"Key":"ClientSecret", "Value":"'"$alibaba_client_secret"'"}
]'
register_credential "alibaba-credential" "ALIBABA" "$ALIBABA_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "alibaba-driver" "alibaba-credential" "ALIBABA" "alibaba"
