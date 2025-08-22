#!/bin/bash
source ./1.export.env

echo "curl -s $OS_COMPUTE_API/os-keypairs/$KEYPAIR_NAME"
curl -s $OS_COMPUTE_API/os-keypairs/$KEYPAIR_NAME -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"
