
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2020.10.28."
echo "##   VM: TerminateVM "
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/vm/${CONN_CONFIG}-vm-001 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

