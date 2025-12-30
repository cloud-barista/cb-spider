#!/bin/bash

# Source common functions
source ./common_register.sh

# Load GCP credentials
GCP_CREDENTIAL_FILE="$HOME/.cb-spider/.gcp-credential"
check_credential_file "$GCP_CREDENTIAL_FILE" "GCP credential file not found"

# Check required variables
check_required_vars "gcp_private_key" "gcp_project_id" "gcp_client_email"

# Register GCP driver
register_driver "gcp-driver" "GCP" "gcp-driver-v1.0.so"

# Register GCP credential
GCP_KEY_VALUES='[
  {"Key":"PrivateKey", "Value":"'"$gcp_private_key"'"},
  {"Key":"ProjectID", "Value":"'"$gcp_project_id"'"},
  {"Key":"ClientEmail", "Value":"'"$gcp_client_email"'"},
  {"Key":"Access Key", "Value":"'"$gcp_access_key"'"},
  {"Key":"Secret", "Value":"'"$gcp_secret"'"}
]'
register_credential "gcp-credential" "GCP" "$GCP_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "gcp-driver" "gcp-credential" "GCP" "gcp"
