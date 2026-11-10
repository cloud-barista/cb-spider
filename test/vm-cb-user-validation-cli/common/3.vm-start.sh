#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)   # this common/ directory (independent of CBSPIDER_ROOT)
source $SETUP_PATH/setup.env $1

source $SETUP_PATH/rest_func.sh

VM_NAME=${VM_NAME}-$2

# "__prefix:<prefix>" => resolve the latest image name (cf. spiderwatch)
IMAGE_NAME=$(resolve_image_name "${IMAGE_NAME}")
if [ -z "$IMAGE_NAME" ]; then
	echo "[FAIL] cannot resolve image name for $1"
	exit 1
fi

echo "============== before start VM: '${VM_NAME}' (image: ${IMAGE_NAME}, spec: ${SPEC_NAME})"

$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD"  vm start -d \
    "{
      \"ConnectionName\":\"${CONN_CONFIG}\",
      \"ReqInfo\": {
        \"Name\": \"${VM_NAME}\",
        \"ImageName\": \"${IMAGE_NAME}\",
        \"VPCName\": \"${VPC_NAME}\",
        \"SubnetName\": \"${SUBNET_NAME}\",
        \"SecurityGroupNames\": [ \"${SG_NAME}\" ],
        \"VMSpecName\": \"${SPEC_NAME}\",
        \"KeyPairName\": \"${KEYPAIR_NAME}\"
      }
    }" 2> /dev/null

echo "============== after start VM: '${VM_NAME}'"

echo -e "\n\n"

