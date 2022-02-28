
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   VPC: Create -> List -> Get"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "vpc-01", "IPv4_CIDR": "10.0.0.0/16", "SubnetInfoList": [ { "Name": "Default-VPC-subnet-2", "IPv4_CIDR": "10.0.12.0/22"} ] } }' |json_pp
curl -sX GET http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

