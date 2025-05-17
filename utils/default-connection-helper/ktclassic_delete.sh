#!/bin/bash

# Source common delete functions
source ./common_delete.sh

# Define variables
DRIVER_NAME="ktclassic-driver"
CREDENTIAL_NAME="ktclassic-credential"
PREFIX="ktclassic"

# Delete connection configs and regions
delete_connection_configs_and_regions "$DRIVER_NAME" "$CREDENTIAL_NAME" "$PREFIX"

# Delete credential
delete_credential "$CREDENTIAL_NAME"

# Delete driver
delete_driver "$DRIVER_NAME"

# Show completion message
show_completion
