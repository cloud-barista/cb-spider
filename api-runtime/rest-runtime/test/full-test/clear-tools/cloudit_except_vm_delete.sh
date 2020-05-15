

echo "####################################################################"
echo "## Full Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   3. KeyPair: CLOUDIT does not support."
echo "##   2. SecurityGroup: Delete"
echo "##   1. VPC: Delete"
echo "####################################################################"


echo "####################################################################"
echo "## 3. KeyPair: CLOUDIT does not support."
echo "####################################################################"

echo "####################################################################"
echo "## 2. SecurityGroup: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/securitygroup/sg-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "####################################################################"
echo "## 1. VPC: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp


