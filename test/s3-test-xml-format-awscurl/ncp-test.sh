#!/bin/bash

# NCP (Naver Cloud Platform) S3 Test Script (SigV4 / awscurl) — Versioning & CORS skipped
# This script sets the connection configuration and runs the S3 API test suite
# excluding versioning and CORS (not supported by NCP Object Storage).
# Author: CB-Spider Team

# Set connection name for NCP
export CONNECTION_NAME="ncp-korea1-config"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the S3 API test (no versioning / no CORS)
echo "Running S3 SigV4 tests with CONNECTION_NAME=$CONNECTION_NAME (no versioning, no CORS)"
"$SCRIPT_DIR/common-s3-api-test-except-versioning-cors.sh"
