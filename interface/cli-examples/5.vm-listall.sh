#!/bin/bash

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' aws|gcp|alibaba|azure|openstack|cloudit'
	echo -e '\n\tex) '$0' aws'
	echo
	exit 0;
fi

source ./setup.env $1

echo "============== before vm listall"
time $CLIPATH/spctl --config $CLIPATH/spctl.conf --cname "${CONN_CONFIG}" vm listall 2> /dev/null
echo "============== after vm listall"
