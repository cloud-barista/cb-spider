#!/bin/bash

# Source common functions
source ./common_register.sh

# Load OpenStack credentials
OPENSTACK_CREDENTIAL_FILE="$HOME/.cb-spider/.openstack-credential"
check_credential_file "$OPENSTACK_CREDENTIAL_FILE" "OpenStack credential file not found"

# Check required variables
check_required_vars "openstack_identity_endpoint" "openstack_username" "openstack_password" "openstack_domain_name" "openstack_project_id"

# Register OpenStack driver
register_driver "openstack-driver" "OPENSTACK" "openstack-driver-v1.0.so"

# Register OpenStack credential
OPENSTACK_KEY_VALUES='[
  {"Key":"IdentityEndpoint", "Value":"'"$openstack_identity_endpoint"'"},
  {"Key":"Username", "Value":"'"$openstack_username"'"},
  {"Key":"Password", "Value":"'"$openstack_password"'"},
  {"Key":"DomainName", "Value":"'"$openstack_domain_name"'"},
  {"Key":"ProjectID", "Value":"'"$openstack_project_id"'"},  
  {"Key":"access", "Value":"'"$openstack_access"'"},
  {"Key":"secret", "Value":"'"$openstack_secret"'"}
]'
register_credential "openstack-credential" "OPENSTACK" "$OPENSTACK_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "openstack-driver" "openstack-credential" "OPENSTACK" "openstack"
