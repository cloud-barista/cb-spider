#!/bin/bash
source ./1.export.env

echo "curl -s $OS_COMPUTE_API/flavors/detail"
curl -s $OS_COMPUTE_API/flavors/detail -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"
