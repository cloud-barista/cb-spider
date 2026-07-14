#!/bin/bash
source ./1.export.env

echo "curl -s $OS_NETWORK_API/firewall/policy"
curl -s $OS_NETWORK_API/firewall/policy --header "X-Auth-Token: $OS_TOKEN"
echo -e "\n"
