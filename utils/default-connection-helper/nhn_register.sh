#!/bin/bash

# Source common functions
source ./common_register.sh

# Load NHN Cloud credentials
NHN_CREDENTIAL_FILE="$HOME/.cb-spider/.nhn-credential"
check_credential_file "$NHN_CREDENTIAL_FILE" "NHN Cloud credential file not found"

# Check required variables
check_required_vars "nhn_identity_endpoint" "nhn_username" "nhn_password" "nhn_domain_name" "nhn_tenant_id"

# Register NHN Cloud driver
register_driver "nhn-driver" "NHN" "nhn-driver-v1.0.so"

# Register NHN Cloud credential
NHN_KEY_VALUES='[
  {"Key":"IdentityEndpoint", "Value":"'"$nhn_identity_endpoint"'"},
  {"Key":"Username", "Value":"'"$nhn_username"'"},
  {"Key":"Password", "Value":"'"$nhn_password"'"},
  {"Key":"DomainName", "Value":"'"$nhn_domain_name"'"},
  {"Key":"TenantId", "Value":"'"$nhn_tenant_id"'"}
]'
register_credential "nhn-credential" "NHN" "$NHN_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "nhn-driver" "nhn-credential" "NHN" "nhn"
