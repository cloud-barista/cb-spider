#!/bin/bash

# Source common functions
source ./common_register.sh

# Load KT Cloud VPC credentials
KTCLOUD_CREDENTIAL_FILE="./.ktvpc-credential"
check_credential_file "$KTCLOUD_CREDENTIAL_FILE" "KT Cloud VPC credential file not found"

# Check required variables
check_required_vars "ktcloudvpc_identity_endpoint" "ktcloudvpc_username" "ktcloudvpc_password" "ktcloudvpc_domain_name" "ktcloudvpc_project_id"

# Register KT Cloud VPC driver
register_driver "ktvpc-driver" "KTCLOUDVPC" "ktcloudvpc-driver-v1.0.so"

# Register KT Cloud VPC credential
KTCLOUD_KEY_VALUES='[
  {"Key":"IdentityEndpoint", "Value":"'"$ktcloudvpc_identity_endpoint"'"},
  {"Key":"Username", "Value":"'"$ktcloudvpc_username"'"},
  {"Key":"Password", "Value":"'"$ktcloudvpc_password"'"},
  {"Key":"DomainName", "Value":"'"$ktcloudvpc_domain_name"'"},
  {"Key":"ProjectID", "Value":"'"$ktcloudvpc_project_id"'"}
]'
register_credential "ktvpc-credential" "KTCLOUDVPC" "$KTCLOUD_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "ktvpc-driver" "ktvpc-credential" "KTCLOUDVPC" "ktvpc"
