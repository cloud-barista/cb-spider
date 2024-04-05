#!/bin/bash
source ./1.export.env_mine

echo "### OS_LB_API - LB List"
echo "curl -v $OS_LB_API?command=listLoadBalancers&zoneid=DX-M1&response=json"
# echo "curl -v $OS_LB_API?command=listLoadBalancers&zoneid=DX-M1&response=json"

curl -v -H "X-Auth-Token: $OS_TOKEN" $OS_LB_API?command=listLoadBalancers

# curl -v -H "X-Auth-Token: $OS_TOKEN" $OS_LB_API?command=listLoadBalancers&zoneid=DX-M1&response=json
# 이것도 동일한 결과
# => &response=json 않붙여도됨.

echo -e "\n"
