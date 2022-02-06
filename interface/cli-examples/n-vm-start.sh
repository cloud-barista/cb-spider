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

	echo "============== before start VM: '${VM_NAME}'"
	time $CLIPATH/spctl --config $CLIPATH/spctl.conf vm start -i json -d \
	    '{
	      "ConnectionName":"'${CONN_CONFIG}'",
	      "ReqInfo": {
		"Name": "'${VM_NAME}'",
		"ImageName": "'${IMAGE_NAME}'",
		"VPCName": "'${VPC_NAME}'",
		"SubnetName": "'${SUBNET_NAME}'",
		"SecurityGroupNames": [ "'${SG_NAME}'" ],
		"VMSpecName": "'${SPEC_NAME}'",
		"KeyPairName": "'${KEYPAIR_NAME}'"
	      }
	    }' 2> /dev/null &

	echo "============== after start VM: '${VM_NAME}'"
done

echo StartVM: Total elapsed time: wait....
time wait $(jobs -p)

