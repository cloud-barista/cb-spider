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

source common/setup.env
source common/$1/setup.env

vminfo=`curl -sX GET http://localhost:1024/spider/vm/$2 -H 'Content-Type: application/json' -d \
	'{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp`
public_ip=`echo -e "$vminfo" |grep \"PublicIP\" |sed -e 's/"PublicIP" : "//g' | sed -e 's/",//g' | sed -e 's/"//g' | sed -e 's/ //g'`

ssh-keygen -f "/home/ubuntu/.ssh/known_hosts" -R $public_ip

#### install nginx
ssh -i ./3.key-test/$KEYPAIR_NAME.pem -o StrictHostKeyChecking=no cb-user@$public_ip "sudo apt-get update"
ssh -i ./3.key-test/$KEYPAIR_NAME.pem -o StrictHostKeyChecking=no cb-user@$public_ip "sudo apt-get install -y nginx"


### setup index.html with public ip

ssh -i ./3.key-test/$KEYPAIR_NAME.pem -o StrictHostKeyChecking=no cb-user@$public_ip "sudo sed -i 's/nginx\!/'"${public_ip}"'/g' /var/www/html/index.nginx-debian.html"
