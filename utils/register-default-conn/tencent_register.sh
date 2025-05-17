#!/bin/bash

# Source common functions
source ./common_register.sh

# Load Tencent credentials
TENCENT_CREDENTIAL_FILE="./.tencent-credential"
check_credential_file "$TENCENT_CREDENTIAL_FILE" "Tencent credential file not found"

# Check required variables
check_required_vars "tencent_secret_id" "tencent_secret_key"

# Register Tencent driver
register_driver "tencent-driver" "TENCENT" "tencent-driver-v1.0.so"

# Register Tencent credential
TENCENT_KEY_VALUES='[
  {"Key":"SecretId", "Value":"'"$tencent_secret_id"'"},
  {"Key":"SecretKey", "Value":"'"$tencent_secret_key"'"}
]'
register_credential "tencent-credential" "TENCENT" "$TENCENT_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "tencent-driver" "tencent-credential" "TENCENT" "tencent"
