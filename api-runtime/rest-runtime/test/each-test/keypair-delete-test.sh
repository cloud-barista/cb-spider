
echo "####################################################################"
echo "## KeyPair Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   KeyPair: Delete"
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/keypair/KEYPAIR-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
