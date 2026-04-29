#!/bin/bash

# Azure S3 Test Script (SigV4 / awscurl)
# This script sets the connection configuration and runs the S3 API test suite
# Azure does not support: Versioning, CORS, Multipart Upload, Delete Marker
# Author: CB-Spider Team

# Set connection name for Azure
export CONNECTION_NAME="azure-northeu-config"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the S3 SigV4 test (no versioning, no CORS, no multipart)
echo "Running S3 SigV4 tests with CONNECTION_NAME=$CONNECTION_NAME (no versioning, no CORS, no multipart)"
export SKIP_MULTIPART=true
export PRESIGNED_UPLOAD_EXTRA_HEADER="x-ms-blob-type: BlockBlob"
"$SCRIPT_DIR/common-s3-api-test-except-versioning-cors.sh"
