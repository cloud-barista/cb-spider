#!/bin/bash
export CSP_NAME="GCP"
export CONNECTION_NAME="${CB_CONNECTION_NAME:-gcp-iowa-config}"
export PUBLICIP_NAME="${CB_PUBLICIP_NAME:-spider-eip-01}"
export RESULT_FILE="${RESULT_DIR:-/tmp/publicip_results_$$}/result_gcp.txt"
# export NIC_NAME=""   # set to run Associate/Disassociate test (use "vmName/nic0" format)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-publicip-test.sh"
