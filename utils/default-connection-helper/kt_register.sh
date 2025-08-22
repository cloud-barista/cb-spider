#!/bin/bash

# Source common functions
source ./common_register.sh

# Load KT Cloud credentials
KTCLOUD_CREDENTIAL_FILE="$HOME/.cb-spider/.kt-credential"
check_credential_file "$KTCLOUD_CREDENTIAL_FILE" "KT Cloud credential file not found"

# Check required variables
check_required_vars "kt_identity_endpoint" "kt_username" "kt_password" "kt_domain_name" "kt_project_id"

# Register KT Cloud driver
register_driver "kt-driver" "KT" "kt-driver-v1.0.so"

# Register KT Cloud credential
KTCLOUD_KEY_VALUES='[
  {"Key":"IdentityEndpoint", "Value":"'"$kt_identity_endpoint"'"},
  {"Key":"Username", "Value":"'"$kt_username"'"},
  {"Key":"Password", "Value":"'"$kt_password"'"},
  {"Key":"DomainName", "Value":"'"$kt_domain_name"'"},
  {"Key":"ProjectID", "Value":"'"$kt_project_id"'"}
]'
register_credential "kt-credential" "KT" "$KTCLOUD_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "kt-driver" "kt-credential" "KT" "kt"
