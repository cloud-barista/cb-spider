#!/bin/bash
source ./1.export.env

echo "### OS_LB_API - LB VM List (New API - Since 20251220)"
echo "curl -v $OS_LB_API/30540a4f-3c1b-4053-932e-e507f2c3d808/servers"

curl -v -H "X-Auth-Token: $OS_TOKEN" $OS_LB_API/30540a4f-3c1b-4053-932e-e507f2c3d808/servers

echo -e "\n"

# (For old NLB API)
# 온라인 고객센터로부터 안내된 예: 
# <GET http://14.63.254.107/openstack4kt/service1/loadbalancer/api?command=listLoadBalancerWebServers&loadbalancerid=38288&multi_stack_id=11&response=json,[x-auth-token:"XXXXXXXXXXXXXXXXXXXX"
