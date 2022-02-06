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
	time $CLIPATH/spctl --config $CLIPATH/spctl.conf --cname "${CONN_CONFIG}" vm list |grep "vm"
	echo
	sleep 1
done
