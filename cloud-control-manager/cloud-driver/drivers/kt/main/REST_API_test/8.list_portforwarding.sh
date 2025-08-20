#!/bin/bash
source ./1.export.env

echo "### OS_NETWORK_API - Portforwarding Rule List"
curl -s $OS_NETWORK_API/Portforwarding -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"
