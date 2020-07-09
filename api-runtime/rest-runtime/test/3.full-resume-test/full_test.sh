
echo "####################################################################"
echo "##   4. VM: Resume"
echo "####################################################################"

curl -sX GET http://localhost:1024/spider/controlvm/${CONN_CONFIG}-vm-01?action=resume -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
