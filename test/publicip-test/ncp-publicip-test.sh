#!/bin/bash
export CSP_NAME="NCP"
export CONNECTION_NAME="${CB_CONNECTION_NAME:-ncp-korea1-config}"
export PUBLICIP_NAME="${CB_PUBLICIP_NAME:-spider-publicip-01}"
export RESULT_FILE="${RESULT_DIR:-/tmp/publicip_results_$$}/result_ncp.txt"
# export VM_NAME=""    # NCP uses VM-level association (not NIC)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-publicip-test.sh"
