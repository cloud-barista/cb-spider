API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD


echo "####################################################################"
echo "##  ${CONN_CONFIG} - VM: Terminate(Delete)"
echo "####################################################################"
curl -u $API_USERNAME:$API_PASSWORD -sX DELETE http://localhost:1024/spider/vm/${CONN_CONFIG}-vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

