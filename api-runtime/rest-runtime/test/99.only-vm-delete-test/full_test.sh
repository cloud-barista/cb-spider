
echo "####################################################################"
echo "##   4. VM: Terminate(Delete)"
echo "####################################################################"

echo "####################################################################"
echo "## 4. VM: Terminate(Delete)"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vm/${CONN_CONFIG}-vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

