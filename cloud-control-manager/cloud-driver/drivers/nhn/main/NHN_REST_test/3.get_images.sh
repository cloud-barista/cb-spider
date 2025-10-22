#!/bin/bash
source ./1.export.env

#curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_IMAGE_API/$TENANT_ID/images  # (X)

curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_IMAGE_API/images
