#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "###########################################################"
echo -e "# 1.create: VPC/Subnet => SG01 => Keypair(save private key) => Start VM "
echo -e "# 2.run: TCP Server and UDP Server"
echo -e "###########################################################"

source ../common/setup.env $1
source setup.env $1


### 1.create: VPC/Subnet => SG01 => Keypair(save private key) => Start VM 
../common/1.prepare-resources.sh $1
../common/3.vm-start.sh $1


### 2.run: TCP Server and UDP Server
echo -e "# check VM status until 'Running'"

for (( num=1; num <= 120; num++ ))
do
        ret=`../common/6.vm-get-status.sh $1`
        result=`echo -e "$ret" | grep Status`
        if [ "$result" == "Status: Running" ];then
                echo -e "# run tcp server and udp server on the target VM"
                P_IP=`../common/./6.vm-get.sh $1 |grep PublicIP: |awk '{print $2}'`
		if [ "$P_IP" ];then
			ssh-keygen -f "$HOME/.ssh/known_hosts" -R "$P_IP" 1> /dev/null 2> /dev/null;
		else
			echo -e ">>>>>>>>>>>>>>> VM's Public IP is NULL!! <<<<<<<<<<<<<<<<"
			exit 0;
		fi
#---- waiting 22 port readiness
                for (( i=1; i <= 120; i++ ))
                do
                        ret1=`nc -zv $P_IP 22 2>&1 | grep succeeded`
                        if [ "$ret1" ];then
                                break;
                        else
                                echo "Trial $i: waiting 22 port readiness";
                                sleep 1;
                        fi
                done

#---- waiting ssh service readiness
                for (( i=1; i <= 120; i++ ))
                do
                        ret2=`ssh -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "hostname" 2>&1 | grep closed`
                        if [ "$ret2" ];then
                                sleep 1;
                                echo "Trial $i: ssh closed => waiting ssh service readiness";
                                continue;
                        else
                                echo "";
                        fi

                        ret3=`ssh -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "hostname" 2>&1 | grep denied`
                        if [ "$ret3" ];then
                                echo "Trial $i: ssh denied => waiting ssh service readiness";
                                sleep 1;
                        else
                                break;
                        fi
                done

                ssh -f -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "sudo nc -vktl 1000"
                ssh -f -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "sudo nc -vkul 2000"

		echo -e "# clear nc processes on the client(this node)"
		sudo killall nc 2> /dev/null

                echo -e "# run tcp server and udp server on the client(this node)"
                sudo nc -vktl 1000&
                sudo nc -vkul 2000&

		sleep 1

		echo -e "\n\n"
		echo -e "###########################################################"
		echo -e "# Prepared All Resources for testing... "
		echo -e "###########################################################"
		echo -e "\n\n"

		exit 0;
        else
                echo -e "# Wait VM Status until Running..."
                sleep 1;
        fi
done

$(error_msg $1 "VM Status is not Running")

echo -e "\n\n"

