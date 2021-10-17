
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   VPC: Register -> List -> Get"
echo "####################################################################"

curl -sX PUT http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "regsiter-vpc-01", "CSPId": "'${VPC_CSPID}'"} }' |json_pp


#curl -sX GET http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

#curl -sX GET http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

