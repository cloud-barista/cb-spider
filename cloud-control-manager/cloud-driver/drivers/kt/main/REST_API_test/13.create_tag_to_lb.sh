#!/bin/bash
source ./1.export.env

echo "### OS_LB_API - Creatge Tag to LB"
echo "curl -v $OS_LB_API?command=createTag&loadbalancerid=41268&tag=123.0.0.0"

curl -v -H "X-Auth-Token: $OS_TOKEN" $OS_LB_API?command=createTag&loadbalancerid=41268&tag=123.0.0.0&response=json

# 뒤에 &response=json 않붙여도됨.

echo -e "\n"
