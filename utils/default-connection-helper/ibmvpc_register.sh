#!/bin/bash

# Source common functions
source ./common_register.sh

# Load IBM credentials
IBM_CREDENTIAL_FILE="./.ibmvpc-credential"
check_credential_file "$IBM_CREDENTIAL_FILE" "IBM credential file not found"

# Check required variables
check_required_vars "ibm_api_key"

# Register IBM driver
register_driver "ibm-driver" "IBM" "ibm-driver-v1.0.so"

# Register IBM credential
IBM_KEY_VALUES='[
  {"Key":"ApiKey", "Value":"'"$ibm_api_key"'"}
]'
register_credential "ibm-credential" "IBM" "$IBM_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "ibm-driver" "ibm-credential" "IBM" "ibm"
