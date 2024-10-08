#!/bin/bash

echo "####################################################################"
echo "##   4. VM: Terminate(Delete)"
echo "##   3. KeyPair: Delete"
echo "##   2. SecurityGroup: Delete"
echo "##   1. VPC: Delete"
echo "####################################################################"

# 4. VM: Terminate(Delete)
echo
echo "####################################################################"
echo "## 4. VM: Terminate(Delete)"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vm/${CONN_CONFIG}-vm-01 \
    -H 'Content-Type: application/json' \
    -d '{ 
        "ConnectionName": "'${CONN_CONFIG}'" 
    }' | json_pp

# 3. KeyPair: Delete
echo
echo "####################################################################"
echo "## 3. KeyPair: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/keypair/keypair-01 \
    -H 'Content-Type: application/json' \
    -d '{ 
        "ConnectionName": "'${CONN_CONFIG}'" 
    }' | json_pp

# 2. SecurityGroup: Delete
echo
echo "####################################################################"
echo "## 2. SecurityGroup: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/securitygroup/sg-01 \
    -H 'Content-Type: application/json' \
    -d '{ 
        "ConnectionName": "'${CONN_CONFIG}'" 
    }' | json_pp

# 1. VPC: Delete
echo
echo "####################################################################"
echo "## 1. VPC: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vpc/vpc-01 \
    -H 'Content-Type: application/json' \
    -d '{ 
        "ConnectionName": "'${CONN_CONFIG}'" 
    }' | json_pp

