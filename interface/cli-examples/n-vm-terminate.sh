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

echo "####################################################################"
echo "## VM: multiple StartVM($max)"
echo "####################################################################"
for (( num=1; num <= $max; num++ ))
do

	VM_NAME=${ORG_VM_NAME}-${num}

	echo "============== before terminate VM: '${VM_NAME}'"
	time $CLIPATH/spiderctl --config $CLIPATH/grpc_conf.yaml --cname "${CONN_CONFIG}" vm terminate -n "${VM_NAME}" 2> /dev/null &

	echo "============== after terminate VM: '${VM_NAME}'"
done

