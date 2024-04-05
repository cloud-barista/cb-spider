#!/bin/bash
source ./1.export.env

# echo "### OS_COMPUTE_API"
# curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_COMPUTE_API/images/6$CREATED_IMAGE_ID
# echo -e "\n\n"

echo "### OS_IMAGE_API - Get created image info."
echo "curl -s $OS_IMAGE_API/images/$CREATED_IMAGE_ID"
curl -s $OS_IMAGE_API/images/$CREATED_IMAGE_ID -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"
