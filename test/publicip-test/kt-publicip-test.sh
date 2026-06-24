#!/bin/bash
export CSP_NAME="KT"
export CONNECTION_NAME="${CB_CONNECTION_NAME:-kt-mokdong1-config}"
export PUBLICIP_NAME="${CB_PUBLICIP_NAME:-spider-publicip-01}"
export RESULT_FILE="${RESULT_DIR:-/tmp/publicip_results_$$}/result_kt.txt"
# export VM_NAME=""    # KT uses VM-level association

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-publicip-test.sh"
