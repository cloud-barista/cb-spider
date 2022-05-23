#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

if [ "$2" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud number'
        echo -e '\n\tex) '$0' aws 5'
        echo
        exit 0;
fi

source ../common/setup.env $1
source setup.env $1


	P_IP=`../common/./6.vm-get.sh $1 $2|grep PublicIP: |awk '{print $2}'`

	echo -e "# Try to terminate test VM"
	for (( i=1; i <= 120; i++ ))
	do
		ret=`../common/7.vm-terminate.sh $1 $2`
		echo -e "$ret"

		result=`echo -e "$ret" |grep "does not exist"`
		if [ "$result" ];then
			ssh-keygen -f "$HOME/.ssh/known_hosts" -R "$P_IP" 1> /dev/null 2> /dev/null;
			break;
		else
			sleep 2
		fi
	done

	echo -e "\n\n"
	echo -e "###########################################################"
	echo -e "# ${VM_NAME}-$2 was Terminated VM "
	echo -e "# ${VM_NAME}-$2 was Terminated VM " >> $TERMINATED_FNAME
	echo -e "###########################################################"

echo -e "\n\n"

