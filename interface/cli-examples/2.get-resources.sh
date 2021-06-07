#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' aws|gcp|alibaba|azure|openstack|cloudit'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

source ./setup.env $1

echo "============== before get KeyPair: '${KEYPAIR_NAME}'"
time $CLIPATH/spctl --config $CLIPATH/grpc_conf.yaml keypair get --cname "${CONN_CONFIG}" -n "${KEYPAIR_NAME}" 2> /dev/null
echo "============== after get KeyPair: '${KEYPAIR_NAME}'"

echo "============== before get SecurityGroup: '${SG_NAME}'"
time $CLIPATH/spctl --config $CLIPATH/grpc_conf.yaml security get --cname "${CONN_CONFIG}" -n "${SG_NAME}" 2> /dev/null
echo "============== after get SecurityGroup: '${SG_NAME}'"

echo "============== before get VPC/Subnet: '${VPC_NAME}'"
time $CLIPATH/spctl --config $CLIPATH/grpc_conf.yaml vpc get --cname "${CONN_CONFIG}" -n "${VPC_NAME}" 2> /dev/null
echo "============== after get VPC/Subnet: '${VPC_NAME}'"

