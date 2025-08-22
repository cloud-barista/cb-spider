#!/bin/bash
source ./1.export.env

# echo "### OS_COMPUTE_API"
# curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_COMPUTE_API/images/$IMAGE_ID
# echo -e "\n\n"

echo "### OS_IMAGE_API - Get image info."
echo "curl -s $OS_IMAGE_API/images/$IMAGE_ID"
curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_IMAGE_API/images/$IMAGE_ID
# curl -s $OS_IMAGE_API/images/$IMAGE_ID -H "X-Auth-Token: $OS_TOKEN"
# 이것도 정상 동작함.

echo -e "\n"
