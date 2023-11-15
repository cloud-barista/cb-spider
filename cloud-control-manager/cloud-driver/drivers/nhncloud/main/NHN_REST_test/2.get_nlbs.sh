#!/bin/bash
source ./1.export.env

# curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_NETWORK_API/$TENANT_ID/lbaas/loadbalancers  : (X) Caution!!

echo -e "\n### LoadBalancers"
curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_NETWORK_API/v2.0/lbaas/loadbalancers

echo -e "\n\n### Listeners"
curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_NETWORK_API/v2.0/lbaas/listeners
echo -e "\n"