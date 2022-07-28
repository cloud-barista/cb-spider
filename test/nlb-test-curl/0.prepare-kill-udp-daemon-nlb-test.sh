#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud nlb_port_number'
        echo -e '\n\tex) '$0' aws vm-01'
        echo
        exit 0;
fi

if [ "$2" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud nlb_port_number'
        echo -e '\n\tex) '$0' aws vm-01'
        echo
        exit 0;
fi

source $1/setup.env

vminfo=`curl -sX GET http://localhost:1024/spider/vm/$2 -H 'Content-Type: application/json' -d \
	'{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp`

public_ip=`echo -e "$vminfo" |grep \"PublicIP\" |sed -e 's/"PublicIP" : "//g' | sed -e 's/",//g' | sed -e 's/"//g' | sed -e 's/ //g'`

#### start udp daemon
ssh -f -i $1-keypair-01.pem -o StrictHostKeyChecking=no cb-user@$public_ip "sudo killall ncat"

