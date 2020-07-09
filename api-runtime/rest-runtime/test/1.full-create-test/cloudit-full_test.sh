
echo "####################################################################"
echo "## Full Test Scripts for CB-Spider IID Working Version - 2020.04.22."
echo "##   1. VPC: Create -> List -> Get"
echo "##   2. SecurityGroup: Create -> List -> Get"
echo "##   3. KeyPair:  CLOUDIT does not support."
echo "##   4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"

echo "####################################################################"
echo "## 1. VPC: Create -> List -> Get"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "vpc-01", "IPv4_CIDR": "192.168.0.0/16", "SubnetInfoList": [ { "Name": "Default-VPC-subnet-2", "IPv4_CIDR": "192.168.1.0/24"} ] } }' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 2. SecurityGroup: Create -> List -> Get"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "sg-01", "VPCName": "vpc-01", "SecurityRules": [ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound"} ] } }' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 3. KeyPair: CLOUDIT does not support."
echo "####################################################################"
echo "#-----------------------------"

echo "####################################################################"
echo "## 4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "'${CONN_CONFIG}'-vm-01", "ImageName": "'${IMAGE_NAME}'", "VPCName": "vpc-01", "SubnetName": "Default-VPC-subnet-2", "SecurityGroupNames": [ "sg-01" ], "VMSpecName": "'${SPEC_NAME}'", "VMUserId": "root", "VMUserPasswd": "etriETRI!@" } }' |json_pp

