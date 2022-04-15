#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "#############################################"
echo -e "# start vm => run tcp server & udp server "
echo -e "#############################################"

source ../common/setup.env $1
source setup.env $1

# print the table header of test results
$(test_result_header $1)

../common/3.vm-start.sh $1

echo -e "# check VM status until 'Running'"

for (( num=1; num <= 120; num++ ))
do
	ret=`../common/6.vm-get-status.sh $1`
	result=`echo -e "$ret" | grep Status`
	if [ "$result"=="Status: Running" ];then
		echo -e "# run tcp server and udp server on the VM"
		P_IP=`../common/./6.vm-get.sh aws |grep PublicIP |awk '{print $2}'`
		ssh -f -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "sudo nc -vktl 1000" 2> /dev/null
		ssh -f -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "sudo nc -vkul 2000" 2> /dev/null		

		#                   CSP I:TCP-01 I:TCP-02 I:UDP-01 I:ICMP-01 | O:TCP-01 O:TCP-02 O:UDP-01 O:ICMP-01
		#./io-traffic-test.sh $1    $2      $3        $4       $5           $6       $7       $8      $9
		./io-traffic-test.sh $1    pass    pass      pass    pass          pass     pass     pass    pass

		# print the end mesg of test results
		$(test_result_tailer)


		# to release local calling processe
		ssh -f -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "sudo killall nc" 2> /dev/null

		exit 0;
	else
		echo -e "# Wait VM Status until Running..."
		sleep 1;
	fi
done

$(error_msg $1 "VM Status is not Running")

echo -e "\n\n"

