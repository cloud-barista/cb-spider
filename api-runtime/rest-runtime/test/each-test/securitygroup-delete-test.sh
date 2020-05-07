
echo "####################################################################"
echo "## SecurityGroup Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   SecurityGroup: Delete"
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/securitygroup/SG-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
