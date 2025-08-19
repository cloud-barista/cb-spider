#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

# common setup.env path
SETUP_PATH=$CBSPIDER_ROOT/test/vm-cb-user-validation-cli/common
source $SETUP_PATH/setup.env $1

echo "============== before create VPC/Subnet: '${VPC_NAME}'"
if [ "$1" = "nhn" ]; then
curl -sX POST http://localhost:1024/spider/regvpc \
  -H 'Content-Type: application/json' \
  -d '{
        "ConnectionName": "'${CONN_CONFIG}'",
        "ReqInfo": { 
          "Name": "'${VPC_NAME}'",
          "CSPId": "'${VPC_CSP_ID}'"
        }
      }' | json_pp

curl -sX POST http://localhost:1024/spider/regsubnet \
  -H 'Content-Type: application/json' \
  -d '{
        "ConnectionName": "'${CONN_CONFIG}'",
        "ReqInfo": { 
          "VpcName": "'${VPC_NAME}'",
          "Name": "'${SUBNET_NAME}'",
          "CSPId": "'${SUBNET_CSP_ID}'"
        }
      }' | json_pp

else
    $CLIPATH/spctl  vpc create -d \
    '{
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": {
        "Name": "'${VPC_NAME}'",
        "IPv4_CIDR": "'${VPC_CIDR}'",
        "SubnetInfoList": [
          {
            "Name": "'${SUBNET_NAME}'",
            "IPv4_CIDR": "'${SUBNET_CIDR}'"
          }
        ]
      }
    }' 2> /dev/null
fi

echo "============== after create VPC/Subnet: '${VPC_NAME}'"

echo -e "\n\n"

echo "============== before create SecurityGroup: '${SG_NAME}'"
$CLIPATH/spctl  securitygroup create -d \
    '{
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": {
        "Name": "'${SG_NAME}'",
        "VPCName": "'${VPC_NAME}'",
        "SecurityRules": [
          {
            "Direction" : "inbound",
            "IPProtocol" : "tcp",
            "FromPort": "1",
            "ToPort" : "65535"
          }
        ]
      }
    }' 2> /dev/null
echo "============== after create SecurityGroup: '${SG_NAME}'"

echo -e "\n\n"

echo "============== before create KeyPair: '${KEYPAIR_NAME}'"
ret=`$CLIPATH/spctl  keypair create  -d \
    '{
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": {
        "Name": "'${KEYPAIR_NAME}'"
      }
    }'`

echo -e "$ret"

result=`echo -e "$ret" | grep already`
if [ "$result" ];then
	echo "You already have the private Key."
else
	echo "$ret" | grep PrivateKey | sed 's/  "PrivateKey": "//g' | sed 's/",//g' | sed 's/\\n/\n/g' > ${KEYPAIR_NAME}.pem
	chmod 600 ${KEYPAIR_NAME}.pem
fi


echo "============== after create KeyPair: '${KEYPAIR_NAME}'"

echo -e "\n\n"
