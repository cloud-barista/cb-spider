#!/bin/bash
source ./1.export.env

echo "### OS_NETWORK_API - VPC"
curl -s $OS_NETWORK_API/VPC -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"

echo "### OS_NETWORK_API - Subnet"
curl -s $OS_NETWORK_API/Network -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"