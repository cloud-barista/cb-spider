#!/bin/bash
source ./1.export.env

echo "curl -s $OS_COMPUTE_API/os-keypairs"
curl -s $OS_COMPUTE_API/os-keypairs -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"
