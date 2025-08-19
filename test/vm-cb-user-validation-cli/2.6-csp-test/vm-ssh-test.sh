#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|ncpvpc|nhn'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

source ../common/setup.env $1
source setup.env $1

echo -e "###########################################################"
echo -e "# 1.Start VM: ${VM_NAME}-$2 "
echo -e "###########################################################"

### 1.create: Start VM 
../common/3.vm-start.sh $1 $2

# Check if the first argument contains the word "mock"
if [[ "$1" == *"mock"* ]]; then
    #echo "Argument contains 'mock'. Exiting script."
    exit 0
fi

### 2.validate to login VM with cb-user@SSH
echo -e "# Check VM status until 'Running' and then try to login with cb-user@SSH"

for (( num=1; num <= 40; num++ ))
do
        ret=`../common/6.vm-get-status.sh $1 $2`
        result=`echo -e "$ret" | grep Status`
        if [ -z "$result" ];then
		exit 0;
	fi
        if [ "$result" == "Status: Running" ];then

                P_IP=`../common/./6.vm-get.sh $1 $2|grep PublicIP: |awk '{print $2}'`
		if [ "$P_IP" ];then
			ssh-keygen -f "$HOME/.ssh/known_hosts" -R "$P_IP" 1> /dev/null 2> /dev/null;
		else
			echo -e ">>>>>>>>>>>>>>> VM's Public IP is NULL!! <<<<<<<<<<<<<<<<"
			exit 0;
		fi
#---- waiting 22 port readiness
                for (( i=1; i <= 40; i++ ))
                do
                        ret1=`nc -zv -w 3 $P_IP 22 2>&1 | grep succeeded`
                        if [ "$ret1" ];then
                                break;
                        else
                                echo "Trial $i: waiting 22 port readiness";
                                sleep 3;
                        fi
                done

#---- waiting ssh service readiness
                for (( i=1; i <= 40; i++ ))
                do
                        ret2=`ssh -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "hostname" 2>&1 | grep closed`
                        if [ "$ret2" ];then
                                sleep 3;
                                echo "Trial $i: ssh closed => waiting ssh service readiness";
                                continue;
                        else
                                echo "";
                        fi

                        ret3=`ssh -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "hostname" 2>&1 | grep denied`
                        if [ "$ret3" ];then
                                echo "Trial $i: ssh denied => waiting ssh service readiness";
                                sleep 3;
                        else
                                break;
                        fi
                done

		ssh -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "hostname" 


		echo -e "###########################################################"
		echo -e "# 2. ${VM_NAME}-$2 was validated!! "
		echo -e "# 2. ${VM_NAME}-$2 was validated!! "  >> $RESULT_FNAME
		echo -e "###########################################################"
		echo -e "\n\n"

		exit 0;
        else
                echo -e "# Wait VM Status until Running..."
                sleep 3;
        fi
done

$(error_msg $1 "VM Status is not Running")

echo -e "\n\n"

