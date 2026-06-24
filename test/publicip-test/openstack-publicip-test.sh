#!/bin/bash
export CSP_NAME="OPENSTACK"
export CONNECTION_NAME="${CB_CONNECTION_NAME:-openstack-config01}"
export PUBLICIP_NAME="${CB_PUBLICIP_NAME:-spider-floatingip-01}"
export RESULT_FILE="${RESULT_DIR:-/tmp/publicip_results_$$}/result_openstack.txt"
# export NIC_NAME=""   # set to run Associate/Disassociate test

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/common-publicip-test.sh"
