#!/bin/bash

# Alibaba S3 Test Script
# This script sets the connection configuration and runs the full S3 API test suite
# Author: CB-Spider Team
# Date: $(date '+%Y-%m-%d %H:%M:%S')

# Set connection name for Alibaba
export CONNECTION_NAME="alibaba-tokyo-config"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the full S3 API test
echo "Running S3 API tests with CONNECTION_NAME=$CONNECTION_NAME"
"$SCRIPT_DIR/common-s3-full-api-test.sh"
