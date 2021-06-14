#!/bin/bash

source ./setup.env $1 $2

max=$2
ORG_VM_NAME=${VM_NAME}

source ./setup.env $1

while true
do
	for (( num=1; num <= $max; num++ ))
	do

		VM_NAME=${ORG_VM_NAME}-${num}

		time $CLIPATH/spctl --config $CLIPATH/grpc_conf.yaml --cname "${CONN_CONFIG}" vm get -n "${VM_NAME}" 2> /dev/null |grep vm
		sleep 10
	done
	echo
	sleep 1
done
