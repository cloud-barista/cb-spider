
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2020.05.19."
echo "##   VM: TerminateVM "
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/vm/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

