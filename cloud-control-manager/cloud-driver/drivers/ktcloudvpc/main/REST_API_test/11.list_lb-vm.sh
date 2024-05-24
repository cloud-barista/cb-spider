#!/bin/bash
source ./1.export.env

echo "### OS_LB_API - LB List"
echo "curl -v $OS_LB_API?command=listLoadBalancerWebServers&loadbalancerid=38288&response=json"

#curl -v -H "X-Auth-Token: $OS_TOKEN" $OS_LB_API?command=listLoadBalancerWebServers&loadbalancerid=38288&response=json
# curl -v -H "X-Auth-Token: $OS_TOKEN" $OS_LB_API?command=listLoadBalancerWebServers&loadbalancerid=38288
#echo -e "\n"

curl -v -H "X-Auth-Token: $OS_TOKEN" $OS_LB_API?command=listLoadBalancerWebServers&loadbalancerid=38288&multi_stack_id=11&response=json

echo -e "\n"

# 온라인 고객센터로부터 안내된 예: 
# <GET http://14.63.254.107/openstack4kt/service1/loadbalancer/api?command=listLoadBalancerWebServers&loadbalancerid=38288&multi_stack_id=11&response=json,[x-auth-token:"gAAAAABlvniyWJCSabJ6CIIDyOIUpZzYRLan_TTXoNEQWmNr1rR3arorAxuLKOmftY8C3TimVz-oXQxnc1rcB2rbS8OAc2WwiGlFXpLUxp1ewsPyTq38G659m1UzKis9pOFC43HBIG4-slOlc3V7oftq4Gyf5DZqMuxmh-EG7y9bHjAx0UxRbOY"
