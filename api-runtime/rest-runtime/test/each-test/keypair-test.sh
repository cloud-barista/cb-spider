
echo "####################################################################"
echo "## KeyPair Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   KeyPair: Create -> List -> Get"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/keypair -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "KEYPAIR-01" } }' |json_pp
curl -sX GET http://localhost:1024/spider/keypair -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/keypair/KEYPAIR-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
