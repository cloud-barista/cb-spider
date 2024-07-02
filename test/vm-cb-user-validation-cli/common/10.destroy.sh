#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/vm-cb-user-validation-cli/common
source $SETUP_PATH/setup.env $1

if [ "$1" = "nhncloud" ]; then
curl -sX DELETE http://localhost:1024/spider/regsubnet/SPIDER-VM-SSH-Test-subnet-01 \
  -H 'Content-Type: application/json' \
  -d '{
        "ConnectionName": "'${CONN_CONFIG}'",
        "ReqInfo": { 
          "VpcName": "'${VPC_NAME}'"
        }
      }' | json_pp

curl -sX DELETE http://localhost:1024/spider/regvpc/SPIDER-VM-SSH-Test-vpc-01 \
  -H 'Content-Type: application/json' \
  -d '{
        "ConnectionName": "'${CONN_CONFIG}'"
      }' | json_pp

fi

# URL to send the DELETE request to
URL="http://localhost:1024/spider/destroy"

CONN=$(cat <<EOF
{
  "ConnectionName": "$CONN_CONFIG"
}
EOF
)

curl -sX DELETE "$URL" \
  -H 'Content-Type: application/json' \
  -d "$CONN" \
  | json_pp
