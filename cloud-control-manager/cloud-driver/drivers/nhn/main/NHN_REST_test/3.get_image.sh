#!/bin/bash
source ./1.export.env

echo "### OS_COMPUTE_API"
curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_COMPUTE_API/$TENANT_ID/images/1cc42a4d-53da-4da9-a464-5241839b68c5

echo -e "\n"
curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_COMPUTE_API/$TENANT_ID/images/1c868787-6207-4ff2-a1e7-ae1331d6829b

#curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_IMAGE_API/$TENANT_ID/images/1c868787-6207-4ff2-a1e7-ae1331d6829ba  # (X)
echo -e "\n\n"

echo "### OS_IMAGE_API"
curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_IMAGE_API/images/1cc42a4d-53da-4da9-a464-5241839b68c5

echo -e "\n"
curl -s -H "X-Auth-Token: $OS_TOKEN" $OS_IMAGE_API/images/1c868787-6207-4ff2-a1e7-ae1331d6829b

echo -e "\n"
