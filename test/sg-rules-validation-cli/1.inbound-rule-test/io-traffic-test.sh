#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "#############################################"
echo -e "# input/output traffic test "
echo -e "#############################################"


source ../common/setup.env $1
source setup.env $1

P_IP=`../common/./6.vm-get.sh $1 |grep PublicIP |awk '{print $2}'`
I_TCP_CMD1="nc -w3 -zvt ${P_IP} 22"
I_TCP_CMD2="nc -w3 -zvt ${P_IP} 1000"
I_UDP_CMD1="nc -w3 -zvu ${P_IP} 2000"
I_ICMP_CMD1="ping -w3 -c3 ${P_IP}"
#---
O_TCP_CMD1="ssh -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP nc -w3 -zvt ${CLIENT1_IP} 22"
O_TCP_CMD2="ssh -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP nc -w3 -zvt ${CLIENT1_IP} 1000"
O_UDP_CMD1="ssh -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP nc -w3 -zvu ${CLIENT1_IP} 2000"
O_ICMP_CMD1="ssh -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP ping -w3 -c3 ${CLIENT1_IP}"



echo -e "\n\n"
echo -e "#-------- CMD List --------#"
echo -e "I_TCP_CMD1= $I_TCP_CMD1"
echo -e "I_TCP_CMD2= $I_TCP_CMD2"
echo -e "I_UDP_CMD1= $I_UDP_CMD1"
echo -e "I_ICMP_CMD1= $I_ICMP_CMD1"
#---
echo -e "O_TCP_CMD1= $O_TCP_CMD1"
echo -e "O_TCP_CMD2= $O_TCP_CMD2"
echo -e "O_UDP_CMD1= $O_UDP_CMD1"
echo -e "O_ICMP_CMD1= $O_ICMP_CMD1"
echo -e "#-------- CMD List --------#"


### expected results mapping
  #                   CSP I:TCP-01 I:TCP-02 I:UDP-01 I:ICMP-01 | O:TCP-01 O:TCP-02 O:UDP-01 O:ICMP-01
  #./io-traffic-test.sh $1    $2      $3        $4       $5           $6       $7       $8      $9
I_TCP_01_EXP="$2"
I_TCP_02_EXP=$3
I_UDP_01_EXP=$4
I_ICMP_01_EXP=$5
#---
O_TCP_01_EXP=$6
O_TCP_02_EXP=$7
O_UDP_01_EXP=$8
O_ICMP_01_EXP=$9


echo -e "\n\n"
echo -e "#================================== INBOUND TEST"
$(new_test)

echo -e "#---------------------- $1:I:TCP-01: VM($P_IP:22) <-- Client"
ret=`$I_TCP_CMD1 2>&1 | grep succeeded`

if [ "$ret" ];then
	$(test_result "$I_TCP_01_EXP" "pass")
else
	$(test_result "$I_TCP_01_EXP" "fail")
fi
#----------------------

echo -e "#---------------------- $1:I:TCP-02: VM($P_IP:1000) <-- Client"
ret=`$I_TCP_CMD2 2>&1 | grep succeeded`

if [ "$ret" ];then
        $(test_result "$I_TCP_02_EXP" "pass")
else
        $(test_result "$I_TCP_02_EXP" "fail")
fi
#----------------------

echo -e "#---------------------- $1:I:UDP-01: VM($P_IP:2000) <-- Client"
if [ "$I_UDP_01_EXP" == "skip" ];then
        $(test_result "skip" "skip")
else
	ret=`$I_UDP_CMD1 2>&1 | grep succeeded`

	if [ "$ret" ];then
		$(test_result "$I_UDP_01_EXP" "pass")
	else
		$(test_result "$I_UDP_01_EXP" "fail")
	fi
fi
#----------------------

echo -e "#---------------------- $1:I:ICMP-01: VM($P_IP:ping) <-- Client"
ret=`$I_ICMP_CMD1 2>&1 | grep icmp_seq`

if [ "$ret" ];then
        $(test_result "$I_ICMP_01_EXP" "pass")
else
        $(test_result "$I_ICMP_01_EXP" "fail")
fi
#----------------------

echo -e "\n\n"


echo -e "#================================== OUTBOUND TEST"

$(test_splitter)

echo -e "#---------------------- $1:O:TCP-01: VM --> Client($CLIENT1_IP:22)"

if [ "$O_TCP_01_EXP" == "skip" ];then
        $(test_result "skip" "skip")
else
	ret=`$O_TCP_CMD1 2>&1 | grep succeeded`

	if [ "$ret" ];then
		$(test_result "$O_TCP_01_EXP" "pass")
	else
		$(test_result "$O_TCP_01_EXP" "fail")
	fi
fi
#----------------------

echo -e "#---------------------- $1:O:TCP-02: VM --> Client($CLIENT1_IP:1000)"

if [ "$O_TCP_02_EXP" == "skip" ];then
        $(test_result "skip" "skip")
else
	ret=`$O_TCP_CMD2 2>&1 | grep succeeded`

	if [ "$ret" ];then
		$(test_result "$O_TCP_02_EXP" "pass")
	else
		$(test_result "$O_TCP_02_EXP" "fail")
	fi
fi
#----------------------

echo -e "#---------------------- $1:O:UDP-01: VM --> Client($CLIENT1_IP:2000)"

if [ "$O_UDP_01_EXP" == "skip" ];then
        $(test_result "skip" "skip")
else
	ret=`$O_UDP_CMD1 2>&1 | grep succeeded`

	if [ "$ret" ];then
		$(test_result "$O_UDP_01_EXP" "pass")
	else
		$(test_result "$O_UDP_01_EXP" "fail")
	fi
fi
#----------------------

echo -e "#---------------------- $1:O:ICMP-01: VM --> Client($CLIENT1_IP:ping)"

if [ "$O_ICMP_01_EXP" == "skip" ];then
        $(test_result "skip" "skip")
else
	ret=`$O_ICMP_CMD1 2>&1 | grep icmp_seq`

	if [ "$ret" ];then
		$(test_result "$O_ICMP_01_EXP" "pass")
	else
		$(test_result "$O_ICMP_01_EXP" "fail")
	fi
fi
#----------------------

echo -e "\n\n"

