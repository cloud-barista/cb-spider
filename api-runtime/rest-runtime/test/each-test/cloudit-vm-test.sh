
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "vm-01", "ImageName": "'${IMAGE_NAME}'", "VPCName": "vpc-01", "SubnetName": "Default-VPC-subnet-2", "SecurityGroupNames": [ "sg-01" ], "VMSpecName": "'${SPEC_NAME}'", "VMUserId": "root", "VMUserPasswd": "etriETRI!@" } }' |json_pp

