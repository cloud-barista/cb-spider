
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   VM: Terminate"
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/vm/VM-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "============== sleep 15 after delete VM"
sleep 15 

