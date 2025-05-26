#!/bin/bash

# Source common functions
source ./common_register.sh

# Load NHN Cloud credentials
NHN_CREDENTIAL_FILE="$HOME/.cb-spider/.nhncloud-credential"
check_credential_file "$NHN_CREDENTIAL_FILE" "NHN Cloud credential file not found"

# Check required variables
check_required_vars "nhncloud_identity_endpoint" "nhncloud_username" "nhncloud_password" "nhncloud_domain_name" "nhncloud_tenant_id"

# Register NHN Cloud driver
register_driver "nhncloud-driver" "NHNCLOUD" "nhncloud-driver-v1.0.so"

# Register NHN Cloud credential
NHN_KEY_VALUES='[
  {"Key":"IdentityEndpoint", "Value":"'"$nhncloud_identity_endpoint"'"},
  {"Key":"Username", "Value":"'"$nhncloud_username"'"},
  {"Key":"Password", "Value":"'"$nhncloud_password"'"},
  {"Key":"DomainName", "Value":"'"$nhncloud_domain_name"'"},
  {"Key":"TenantId", "Value":"'"$nhncloud_tenant_id"'"}
]'
register_credential "nhncloud-credential" "NHNCLOUD" "$NHN_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "nhncloud-driver" "nhncloud-credential" "NHNCLOUD" "nhncloud"
