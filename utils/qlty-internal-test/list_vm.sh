API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD


echo "####################################################################"
echo "##  ${CONN_CONFIG} - VM: ListStatus"
echo "####################################################################"

curl -u $API_USERNAME:$API_PASSWORD -sX GET http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
