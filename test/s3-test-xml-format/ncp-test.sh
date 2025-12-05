#!/bin/bash

# NCP S3 Test Script
# This script sets the connection configuration and runs the S3 API test suite for NCP
# Naver Cloud Platform Object Storage does not support: Versioning and CORS
# Author: CB-Spider Team
# Date: $(date '+%Y-%m-%d %H:%M:%S')

# Set connection name for NCP
export CONNECTION_NAME="ncp-korea1-config"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the S3 API test (excluding Versioning and CORS)
echo "Running S3 API tests with CONNECTION_NAME=$CONNECTION_NAME"
echo "Note: Versioning and CORS tests will be skipped (not supported by NCP)"
"$SCRIPT_DIR/common-s3-api-test-except-versioning-cors.sh"
