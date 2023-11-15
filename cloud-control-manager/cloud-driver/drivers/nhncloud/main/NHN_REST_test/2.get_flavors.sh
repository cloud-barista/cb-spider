#!/bin/bash
source ./1.export.env

curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_COMPUTE_API/$TENANT_ID/flavors
