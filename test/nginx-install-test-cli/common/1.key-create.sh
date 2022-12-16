#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/nginx-install-test-cli/common
source $SETUP_PATH/setup.env $1



echo "============== before create KeyPair: '${KEYPAIR_NAME}'"
ret=`$CLIPATH/spctl --config $CLIPATH/spctl.conf keypair create -i json -o json -d \
    '{
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": {
        "Name": "'${KEYPAIR_NAME}'"
      }
    }'`

echo -e "$ret"

echo "============== after create KeyPair: '${KEYPAIR_NAME}'"

result=`echo -e "$ret" | grep already`
if [ "$result" ];then
        echo "You already have the private Key."
else
        echo "$ret" | grep PrivateKey | sed 's/  "PrivateKey": "//g' | sed 's/",//g' | sed 's/\\n/\n/g' > ${KEYPAIR_NAME}.pem
        chmod 600 ${KEYPAIR_NAME}.pem
fi

echo -e "\n\n"

