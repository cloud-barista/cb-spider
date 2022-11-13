#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud|ncpvpc|ktvpc nlb_port_number'
        echo -e '\n\tex) '$0' aws 22'
        echo
        exit 0;
fi

if [ "$2" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud|ncpvpc|ktvpc nlb_port_number'
        echo -e '\n\tex) '$0' aws 22'
        echo
        exit 0;
fi

source $1/setup.env

nlbinfo=`curl -sX GET http://localhost:1024/spider/nlb/spider-nlb-01 -H 'Content-Type: application/json' -d \
	'{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp`


nlb_ip=`echo -e "$nlbinfo" |grep IP |sed -e 's/"IP" : "//g' | sed -e 's/",//g' | sed -e 's/"//g' | sed -e 's/ //g'`

if [ "$nlb_ip" = "" ]; then
	nlb_ip=`echo -e "$nlbinfo" |grep DNSName |grep -v Key |sed -e 's/"DNSName" : "//g' | sed -e 's/",//g' | sed -e 's/"//g' | sed -e 's/ //g'`
fi

./common/check-nlb-ssh-test.sh $1 $nlb_ip $2
