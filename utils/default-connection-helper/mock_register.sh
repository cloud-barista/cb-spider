#!/bin/bash

# Source common functions
source ./common_register.sh

# Load Mock credentials
MOCK_CREDENTIAL_FILE="./.mock-credential"
check_credential_file "$MOCK_CREDENTIAL_FILE" "Mock credential file not found"

# Check required variables
check_required_vars "mock_name"

# Register Mock driver
register_driver "mock-driver" "MOCK" "mock-driver-v1.0.so"

# Register Mock credential
MOCK_KEY_VALUES='[
  {"Key":"MockName", "Value":"'"$mock_name"'"}
]'
register_credential "mock-credential" "MOCK" "$MOCK_KEY_VALUES"

# Register regions and connection configs
register_regions_and_configs "mock-driver" "mock-credential" "MOCK" "mock"
