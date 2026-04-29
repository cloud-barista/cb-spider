#!/bin/bash

# Azure S3 Test Script (XML format)
# This script sets the connection configuration and runs the S3 API test suite
# Azure does not support: Versioning, CORS, Multipart Upload, Delete Marker
# Author: CB-Spider Team

# Set connection name for Azure
export CONNECTION_NAME="azure-northeu-config"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the S3 API test (excluding Versioning, CORS, and Multipart Upload)
echo "Running S3 API tests with CONNECTION_NAME=$CONNECTION_NAME"
echo "Note: Versioning, CORS, and Multipart Upload tests will be skipped (not supported by Azure)"
export SKIP_MULTIPART=true
export PRESIGNED_UPLOAD_EXTRA_HEADER="x-ms-blob-type: BlockBlob"
"$SCRIPT_DIR/common-s3-api-test-except-versioning-cors.sh"
