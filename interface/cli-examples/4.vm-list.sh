#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' aws|gcp|alibaba|azure|openstack|cloudit'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

source ./setup.env $1

echo "============== before vm list"
time $CLIPATH/spctl --config $CLIPATH/grpc_conf.yaml --cname "${CONN_CONFIG}" vm list 2> /dev/null
echo "============== after vm list"
