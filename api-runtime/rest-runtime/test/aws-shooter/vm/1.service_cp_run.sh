#!/bin/bash
source ../setup.env


CONNECT_NAME=${CONNECT_NAMES[0]}

echo ========================== $CONNECT_NAME
PUBLIC_IPS=`curl -sX GET http://$RESTSERVER:1024/vm?connection_name=$CONNECT_NAME |json_pp |grep "\"PublicIP\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for PUBLIC_IP in ${PUBLIC_IPS}
do
        echo $CONNECT_NAME : copy testsvc into ${PUBLIC_IP} ...
	ssh-keygen -f "/root/.ssh/known_hosts" -R ${PUBLIC_IP}
        scp -i ../keypair/${CONNECT_NAME}.key -o "StrictHostKeyChecking no" ./testsvc/TESTSvc ./testsvc/setup.env ubuntu@$PUBLIC_IP:/tmp
        scp -i ../keypair/${CONNECT_NAME}.key -o "StrictHostKeyChecking no" -r ./testsvc/conf ubuntu@$PUBLIC_IP:/tmp
#        ssh -i ../keypair/${CONNECT_NAME}.key -o "StrictHostKeyChecking no" ubuntu@$PUBLIC_IP 'source /tmp/setup.env;/tmp/TESTSvc'
done

