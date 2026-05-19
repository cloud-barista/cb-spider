#!/bin/bash

# KT Cloud S3 Test Script (SigV4 / awscurl)
# This script sets the connection configuration and runs the full S3 API test suite
# Author: CB-Spider Team

# Set connection name for KT Cloud
export CONNECTION_NAME="kt-mokdong1-config"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the full S3 API test
echo "Running S3 SigV4 tests with CONNECTION_NAME=$CONNECTION_NAME"
"$SCRIPT_DIR/common-s3-full-api-test.sh"
