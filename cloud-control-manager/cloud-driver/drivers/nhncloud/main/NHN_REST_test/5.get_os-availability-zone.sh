#!/bin/bash
source ./1.export.env

# echo "curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_COMPUTE_API/$TENANT_ID/os-availability-zone"
curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_COMPUTE_API/$TENANT_ID/os-availability-zone
