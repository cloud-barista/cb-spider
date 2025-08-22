#!/bin/bash

# Source common delete functions
source ./common_delete.sh

# Define variables
DRIVER_NAME="kt-driver"
CREDENTIAL_NAME="kt-credential"
PREFIX="kt"

# Delete connection configs and regions
delete_connection_configs_and_regions "$DRIVER_NAME" "$CREDENTIAL_NAME" "$PREFIX"

# Delete credential
delete_credential "$CREDENTIAL_NAME"

# Delete driver
delete_driver "$DRIVER_NAME"

# Show completion message
show_completion
