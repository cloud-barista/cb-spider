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
echo "============== before get VM: '${VM_NAME}'"
time $CLIPATH/spctl --config $CLIPATH/spctl.conf --cname "${CONN_CONFIG}" vm get -n "${VM_NAME}" 2> /dev/null
echo "============== after get VM: '${VM_NAME}'"
