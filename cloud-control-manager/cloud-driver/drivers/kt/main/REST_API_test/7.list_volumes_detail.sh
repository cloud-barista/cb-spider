#!/bin/bash
source ./1.export.env

echo "curl -s $OS_VOLUME_API/$PROJECT_ID/volumes/detail"
curl -s $OS_VOLUME_API/$PROJECT_ID/volumes/detail -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"
