#!/bin/bash
source ./1.export.env

echo "curl -s $OS_IMAGE_API/images"
curl -s $OS_IMAGE_API/images -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"
