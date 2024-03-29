
echo "####################################################################"
echo "## Full Test Scripts for CB-Spider IID Working Version - 2020.04.22."
echo "##   1. VPC: Create -> Add-Subnet -> List -> Get"
echo "##   2. SecurityGroup: Create -> List -> Get"
echo "##   3. KeyPair: Create -> List -> Get"
echo "##   4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "## ---------------------------------"
echo "##   4. VM: Terminate(Delete)"
echo "##   3. KeyPair: Delete"
echo "##   2. SecurityGroup: Delete"
echo "##   1. VPC: Remove-Subnet -> Delete"
echo "####################################################################"

echo "####################################################################"
echo "## 1. VPC: Create -> List -> Get"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "vpc-01", "IPv4_CIDR": "192.168.0.0/16", "SubnetInfoList": [ { "Name": "subnet-01", "IPv4_CIDR": "192.168.1.0/24"} ] } }' |json_pp
curl -sX POST http://localhost:1024/spider/vpc/vpc-01/subnet -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "subnet-02", "IPv4_CIDR": "192.168.2.0/24" } }' |json_pp
curl -sX GET http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vpc?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
curl -sX GET http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vpc/vpc-01?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 2. SecurityGroup: Create -> List -> Get"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "sg-01", "VPCName": "vpc-01", "SecurityRules": [ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound", "CIDR" : "0.0.0.0/0"} ] } }' |json_pp
curl -sX GET http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/securitygroup?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
curl -sX GET http://localhost:1024/spider/securitygroup/sg-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/securitygroup/sg-01?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 3. KeyPair: Create -> List -> Get"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/keypair -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "keypair-01" } }' |json_pp
curl -sX GET http://localhost:1024/spider/keypair -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/keypair?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
curl -sX GET http://localhost:1024/spider/keypair/keypair-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/keypair/keypair-01?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "vm-01", "ImageName": "'${IMAGE_NAME}'", "VPCName": "vpc-01", "SubnetName": "subnet-01", "SecurityGroupNames": [ "sg-01" ], "VMSpecName": "'${SPEC_NAME}'", "KeyPairName": "keypair-01"} }' |json_pp
echo "============== sleep 1 after start VM"
sleep 1 
curl -sX GET http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vm?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
echo "============== after List VM"
curl -sX GET http://localhost:1024/spider/vm/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vm/vm-01?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
echo "============== after Get VM"

curl -sX GET http://localhost:1024/spider/vmstatus -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vmstatus?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
echo "============== after List VM Status"
curl -sX GET http://localhost:1024/spider/vmstatus/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vmstatus/vm-01?ConnectionName=${CONN_CONFIG} -H 'Content-Type: application/json' |json_pp
echo "============== after Get VM Status"

curl -sX GET http://localhost:1024/spider/controlvm/vm-01?action=suspend -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/controlvm/vm-01?'ConnectionName='${CONN_CONFIG}'&action=suspend' -H 'Content-Type: application/json' |json_pp
echo "============== sleep 1 after suspend VM"
sleep 1
curl -sX GET http://localhost:1024/spider/controlvm/vm-01?action=resume -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/controlvm/vm-01?'ConnectionName='${CONN_CONFIG}'&action=resume' -H 'Content-Type: application/json' |json_pp
echo "============== sleep 1 after resume VM"
sleep 1
curl -sX GET http://localhost:1024/spider/controlvm/vm-01?action=reboot -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/controlvm/vm-01?'ConnectionName='${CONN_CONFIG}'&action=reboot' -H 'Content-Type: application/json' |json_pp
echo "============== sleep 1 after reboot VM"
sleep 1 
echo "#-----------------------------"


echo "####################################################################"
echo "####################################################################"
echo "####################################################################"

echo "####################################################################"
echo "## 4. VM: Terminate(Delete)"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vm/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "============== sleep 1 after delete VM"
sleep 1 

echo "####################################################################"
echo "## 3. KeyPair: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/keypair/keypair-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "####################################################################"
echo "## 2. SecurityGroup: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/securitygroup/sg-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "####################################################################"
echo "## 1. VPC: Remove-Subnet -> Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vpc/vpc-01/subnet/subnet-02 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX DELETE http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

