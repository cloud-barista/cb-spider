#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "#############################################"
echo -e "# TEST: $0 "
echo -e "#############################################"

source ../common/setup.env $1
source setup.env $1

echo "============== before list SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl -u "$API_USERNAME" -p "$API_PASSWORD" securitygroup list -c "${CONN_CONFIG}" 2> /dev/null
echo "============== after list SecurityGroup: '${SG_NAME}'"

echo "============== before get SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl -u "$API_USERNAME" -p "$API_PASSWORD" securitygroup get -c "${CONN_CONFIG}" -n "${SG_NAME}" 2> /dev/null
echo "============== after get SecurityGroup: '${SG_NAME}'"

