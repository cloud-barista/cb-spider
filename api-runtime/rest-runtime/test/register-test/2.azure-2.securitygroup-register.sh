export CONN_CONFIG=azure-northeu-config


export VPC_NAME=vpc-01
export SG_NAME=sg-01
export SG_CSPID=/subscriptions/a20fed83-96bd-4480-92a9-140b8e3b7c3a/resourceGroups/cb-group-wip/providers/Microsoft.Network/networkSecurityGroups/powerkim-test

./securitygroup-register.sh
