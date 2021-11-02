echo "#### Full Test Process - Start ###"

sleep 1 

echo "####################################################################"
echo "## Full Test Scripts for CB-Spider IID Working Version - 2020.04.22."
echo "##   0. VM Image: List"
echo "##   0. VM Spec: List"
echo "##   1. VPC: Create -> List -> Get"
echo "##   2. SecurityGroup: Create -> List -> Get"
echo "##   3. KeyPair: Create -> List -> Get"
echo "##   4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "## ---------------------------------"
echo "##   4. VM: Terminate(Delete)"
echo "##   3. KeyPair: Delete"
echo "##   2. SecurityGroup: Delete"
echo "##   1. VPC: Delete"
echo "####################################################################"

sleep 2 

echo "####################################################################"
echo "##   0. VM Image: List"
echo "####################################################################"

curl -sX GET http://localhost:1024/spider/vmimage -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

sleep 5 

echo "####################################################################"
echo "##   0. VM Spec: List"
echo "####################################################################"

curl -sX GET http://localhost:1024/spider/vmspec -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

sleep 5 

echo "####################################################################"
echo "## 1. VPC: Create -> List -> Get"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "Tencent-VPC-01", "IPv4_CIDR": "10.0.0.0/16", "SubnetInfoList": [ { "Name": "Tencent-Subnet-01", "IPv4_CIDR": "10.0.1.0/24"} ] } }' |json_pp

sleep 5 

curl -sX GET http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

sleep 5 

curl -sX GET http://localhost:1024/spider/vpc/Tencent-VPC-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 2. SecurityGroup: Create -> List -> Get"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "Tencent-SG-01", "VPCName": "Tencent-VPC-01", "SecurityRules": [ {"FromPort": "22", "ToPort" : "22", "IPProtocol" : "tcp", "Direction" : "inbound"} ] } }' |json_pp

sleep 5 

curl -sX GET http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

sleep 5 

curl -sX GET http://localhost:1024/spider/securitygroup/Tencent-SG-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 3. KeyPair: Create -> List -> Get"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/keypair -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "Tencent-Key-01" } }' |json_pp

sleep 5 

curl -sX GET http://localhost:1024/spider/keypair -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

sleep 5

curl -sX GET http://localhost:1024/spider/keypair/Tencent-Key-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "#-----------------------------"

echo "####################################################################"
echo "## 4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "Tencent-VM-01", "ImageName": "'${IMAGE_NAME}'", "VPCName": "Tencent-VPC-01", "SubnetName": "Tencent-Subnet-01", "SecurityGroupNames": [ "Tencent-SG-01" ], "VMSpecName": "'${SPEC_NAME}'", "KeyPairName": "Tencent-Key-01"} }' |json_pp

echo "============== sleep 50sec after start VM"
sleep 50 

curl -sX GET http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

echo "============== sleep 10sec after List VM"
sleep 10 

curl -sX GET http://localhost:1024/spider/vm/Tencent-VM-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

echo "============== sleep 10sec after Get VM"
sleep 10 

curl -sX GET http://localhost:1024/spider/vmstatus -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

echo "============== sleep 2sec after List VM Status"
sleep 2 

curl -sX GET http://localhost:1024/spider/vmstatus/Tencent-VM-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

echo "============== sleep 10sec after Get VM Status"
sleep 10

curl -sX GET http://localhost:1024/spider/controlvm/Tencent-VM-01?action=suspend -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

echo "============== sleep 30sec after suspend VM"
sleep 30 

curl -sX GET http://localhost:1024/spider/controlvm/Tencent-VM-01?action=resume -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

echo "============== sleep 10sec after resume VM"
sleep 10

curl -sX GET http://localhost:1024/spider/controlvm/Tencent-VM-01?action=reboot -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

echo "============== sleep 70sec after reboot VM"
sleep 70 
echo "#-----------------------------"


echo "####################################################################"
echo "####################################################################"
echo "####################################################################"

echo "####################################################################"
echo "## 4. VM: Terminate(Delete)"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vm/Tencent-VM-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

echo "============== sleep 50sec after delete VM"
sleep 50


echo "####################################################################"
echo "## 3. KeyPair: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/keypair/Tencent-Key-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

sleep 5

echo "####################################################################"
echo "## 2. SecurityGroup: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/securitygroup/Tencent-SG-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

sleep 5

echo "####################################################################"
echo "## 1. VPC: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vpc/Tencent-VPC-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp


echo "#### Full Test Process - Finished ###"