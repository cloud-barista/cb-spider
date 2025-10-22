#!/bin/bash
source ./1.export.env

echo "curl -s $OS_COMPUTE_API/flavors/$FLAVOR_ID"
curl -s $OS_COMPUTE_API/flavors/$FLAVOR_ID -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"
