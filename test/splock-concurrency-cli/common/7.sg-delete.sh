#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/splock-concurrency-cli/common
source $SETUP_PATH/setup.env $1

SG_NAME=${SG_NAME}-$2

echo "============== before delete SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl -u "$SPIDER_USERNAME" -p "$SPIDER_PASSWORD" securitygroup delete -c "${CONN_CONFIG}" -n "${SG_NAME}" 2> /dev/null
echo "============== after delete SecurityGroup: '${SG_NAME}'"

echo -e "\n\n"

