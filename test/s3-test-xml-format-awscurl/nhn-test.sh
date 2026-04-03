#!/bin/bash

# NHN Cloud S3 Test Script (SigV4 / awscurl) — Versioning & CORS skipped
# This script sets the connection configuration and runs the S3 API test suite
# excluding versioning and CORS (not supported by NHN Object Storage).
# Author: CB-Spider Team

# Set connection name for NHN
export CONNECTION_NAME="nhn-korea-pangyo1-config"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the S3 API test (no versioning / no CORS)
echo "Running S3 SigV4 tests with CONNECTION_NAME=$CONNECTION_NAME (no versioning, no CORS)"
"$SCRIPT_DIR/common-s3-api-test-except-versioning-cors.sh"
