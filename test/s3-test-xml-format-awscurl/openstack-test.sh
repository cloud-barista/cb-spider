#!/bin/bash

# OpenStack Swift S3 Test Script (SigV4 / awscurl) — Multipart & Versioning skipped
# This script sets the connection configuration and runs the S3 API test suite
# excluding multipart upload and versioning (not supported by OpenStack Swift).
# Author: CB-Spider Team

# Set connection name for OpenStack
export CONNECTION_NAME="openstack-config01"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the S3 API test (no multipart / no versioning)
echo "Running S3 SigV4 tests with CONNECTION_NAME=$CONNECTION_NAME (no multipart, no versioning)"
"$SCRIPT_DIR/common-s3-api-test-except-multipart-versioning.sh"
