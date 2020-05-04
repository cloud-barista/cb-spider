
echo "####################################################################"
echo "## SecurityGroup Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   SecurityGroup: Create -> List -> Get"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "SG-01", "VPCName": "VPC-01", "SecurityRules": [ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound"} ] } }' |json_pp
curl -sX GET http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/securitygroup/SG-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
