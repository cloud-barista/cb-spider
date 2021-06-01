#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' aws|gcp|alibaba|azure|openstack|cloudit'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

source ./setup.env $1

VM_NAME=${VM_NAME}-1
echo "============== before terminate VM: '${VM_NAME}'"
time $CLIPATH/spiderctl --config $CLIPATH/grpc_conf.yaml --cname "${CONN_CONFIG}" vm terminate -n "${VM_NAME}" 2> /dev/null
echo "============== after terminate VM: '${VM_NAME}'"

