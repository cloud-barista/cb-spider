#!/bin/bash
source ./1.export.env

echo "### OS_LB_API - Creatge LB"
echo "curl -v $OS_LB_API?command=createLoadBalancer&name=LbDxApi07&healthchecktype=TCP&healthcheckurl=abc.kt.com&loadbalanceroption=roundrobin&serviceport=80&servicetype=TCP&zoneid=DX-M1"

curl -v -H "X-Auth-Token: $OS_TOKEN" $OS_LB_API?command=createLoadBalancer&name=LbDxApi07&healthchecktype=TCP&healthcheckurl=abc.kt.com&loadbalanceroption=roundrobin&name=kt-lb-dx-api-07&serviceport=80&servicetype=TCP&zoneid=DX-M1

# 뒤에 &response=json 않붙여도됨.

echo -e "\n"
