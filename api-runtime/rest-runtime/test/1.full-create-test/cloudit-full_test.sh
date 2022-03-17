
echo "####################################################################"
echo "## Full Test Scripts for CB-Spider IID Working Version - 2020.04.22."
echo "##   1. VPC: Create -> List -> Get"
echo "##   2. SecurityGroup: Create -> List -> Get"
echo "##   3. KeyPair:  CLOUDIT does not support."
echo "##   4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"

echo "####################################################################"
echo "## 1. VPC: Create"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "vpc-01", "IPv4_CIDR": "10.0.0.0/16", "SubnetInfoList": [ { "Name": "subnet-01", "IPv4_CIDR": "10.0.8.0/22"} ] } }' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 2. SecurityGroup: Create"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "sg-01", "VPCName": "vpc-01", "SecurityRules": [ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound"} ] } }' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 3. KeyPair: Create"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/keypair -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "keypair-01" } }' |json_pp
echo "#-----------------------------"


echo "####################################################################"
echo "## 4. VM: StartVM"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "'${CONN_CONFIG}'-vm-01", "ImageName": "'${IMAGE_NAME}'", "VPCName": "vpc-01", "SubnetName": "subnet-01", "SecurityGroupNames": [ "sg-01" ], "VMSpecName": "'${SPEC_NAME}'", "KeyPairName": "keypair-01"}, "VMUserId": "root", "VMUserPasswd": "etriETRI!@" } }' |json_pp

