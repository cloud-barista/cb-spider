#!/bin/bash

# Source common functions
source ./common_register.sh

# Load KT Cloud Classic credentials
KTCLOUD_CREDENTIAL_FILE="./.ktclassic-credential"
check_credential_file "$KTCLOUD_CREDENTIAL_FILE" "KT Cloud Classic credential file not found"

# Check required variables
check_required_vars "ktcloudclassic_client_id" "ktcloudclassic_client_secret"

# Register KT Cloud Classic driver
register_driver "ktclassic-driver" "KTCLOUD" "ktcloud-driver-v1.0.so"

# Register KT Cloud Classic credential
KTCLOUD_KEY_VALUES='[
  {"Key":"ClientId", "Value":"'"$ktcloudclassic_client_id"'"},
  {"Key":"ClientSecret", "Value":"'"$ktcloudclassic_client_secret"'"}
]'
register_credential "ktclassic-credential" "KTCLOUD" "$KTCLOUD_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "ktclassic-driver" "ktclassic-credential" "KTCLOUD" "ktclassic"
