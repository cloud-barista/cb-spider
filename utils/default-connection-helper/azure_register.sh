#!/bin/bash

# Source common functions
source ./common_register.sh

# Load Azure credentials
AZURE_CREDENTIAL_FILE="./.azure-credential"
check_credential_file "$AZURE_CREDENTIAL_FILE" "Azure credential file not found"

# Check required variables
check_required_vars "azure_client_id" "azure_client_secret" "azure_tenant_id" "azure_subscription_id"

# Register Azure driver
register_driver "azure-driver" "AZURE" "azure-driver-v1.0.so"

# Register Azure credential
AZURE_KEY_VALUES='[
  {"Key":"ClientId", "Value":"'"$azure_client_id"'"},
  {"Key":"ClientSecret", "Value":"'"$azure_client_secret"'"},
  {"Key":"TenantId", "Value":"'"$azure_tenant_id"'"},
  {"Key":"SubscriptionId", "Value":"'"$azure_subscription_id"'"}
]'
register_credential "azure-credential" "AZURE" "$AZURE_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "azure-driver" "azure-credential" "AZURE" "azure"
