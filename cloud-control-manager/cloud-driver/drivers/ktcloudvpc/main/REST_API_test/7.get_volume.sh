#!/bin/bash
source ./1.export.env

echo "curl -s $OS_VOLUME_API/$PROJECT_ID/volumes/$VOLUME_ID"
curl -s $OS_VOLUME_API/$PROJECT_ID/volumes/$VOLUME_ID -H "X-Auth-Token: $OS_TOKEN"
echo -e "\n"