#!/bin/bash

# Source common functions
source ./common_register.sh

# Load KT Cloud Classic credentials
KTCLASSIC_CREDENTIAL_FILE="$HOME/.cb-spider/.ktclassic-credential"
check_credential_file "$KTCLASSIC_CREDENTIAL_FILE" "KT Cloud Classic credential file not found"

# Check required variables
check_required_vars "ktclassic_client_id" "ktclassic_client_secret"

# Register KT Cloud Classic driver
register_driver "ktclassic-driver" "KTCLASSIC" "ktclassic-driver-v1.0.so"

# Register KT Cloud Classic credential
KTCLASSIC_KEY_VALUES='[
  {"Key":"ClientId", "Value":"'"$ktclassic_client_id"'"},
  {"Key":"ClientSecret", "Value":"'"$ktclassic_client_secret"'"}
]'
register_credential "ktclassic-credential" "KTCLASSIC" "$KTCLASSIC_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "ktclassic-driver" "ktclassic-credential" "KTCLASSIC" "ktclassic"
