#!/bin/bash

# NHN S3 Test Script - JSON Format
# This script sets the connection configuration and runs the S3 API test suite for NHN
# NHN Cloud Object Storage does not support: Versioning, CORS
# Author: CB-Spider Team
# Date: $(date '+%Y-%m-%d %H:%M:%S')

# Set connection name for NHN
export CONNECTION_NAME="nhn-korea-pangyo1-config"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the S3 API test (excluding Versioning and CORS)
echo "Running S3 API tests (JSON format) with CONNECTION_NAME=$CONNECTION_NAME"
echo "Note: Versioning and CORS tests will be skipped (not supported by NHN)"
"$SCRIPT_DIR/common-s3-api-test-except-versioning-cors.sh"
