#!/bin/bash

# Tencent S3 Test Script
# This script sets the connection configuration and runs the full S3 API test suite
# Author: CB-Spider Team
# Date: $(date '+%Y-%m-%d %H:%M:%S')

# Set connection name for Tencent
#export CONNECTION_NAME="tencent-beijing3-config"
export CONNECTION_NAME="tencent-tokyo-config"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the full S3 API test
echo "Running S3 API tests with CONNECTION_NAME=$CONNECTION_NAME"
"$SCRIPT_DIR/common-s3-full-api-test.sh"
