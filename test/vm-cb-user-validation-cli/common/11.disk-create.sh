#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn|kt'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)   # this common/ directory (independent of CBSPIDER_ROOT)
source $SETUP_PATH/setup.env $1

echo "============== before create Disk: '${DISK_NAME}' (type: ${DISK_TYPE}, size: ${DISK_SIZE})"
$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD" disk create -d \
    "{
      \"ConnectionName\":\"${CONN_CONFIG}\",
      \"ReqInfo\": {
        \"Name\": \"${DISK_NAME}\",
        \"DiskType\": \"${DISK_TYPE}\",
        \"DiskSize\": \"${DISK_SIZE}\"
      }
    }"
echo "============== after create Disk: '${DISK_NAME}'"

echo -e "\n\n"
