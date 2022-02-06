#!/bin/bash

if [ "$1" = "" ] || [ "$2" = "" ]; then
        echo
        echo -e 'usage: '$0' aws|gcp|alibaba|azure|openstack|cloudit # of VMs'
        echo -e '\n\tex) '$0' aws 10'
        echo
        exit 0;
fi

source ./setup.env $1 $2

max=$2
ORG_VM_NAME=${VM_NAME}

while true
do
	for (( num=1; num <= $max; num++ ))
	do

		VM_NAME=${ORG_VM_NAME}-${num}

		time $CLIPATH/spctl --config $CLIPATH/spctl.conf --cname "${CONN_CONFIG}" vm get -n "${VM_NAME}" 2> /dev/null |grep vm
		#sleep 1
	done
	echo
	sleep 2
done
