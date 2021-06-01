#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' aws|gcp|alibaba|azure|openstack|cloudit'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

source ./setup.env $1

while true
do
	echo "============== vm list =============="
	time $CLIPATH/spiderctl --config $CLIPATH/grpc_conf.yaml --cname "${CONN_CONFIG}" vm list |grep "NameId: powerkim-vm-01"
	echo
	sleep 1
done
