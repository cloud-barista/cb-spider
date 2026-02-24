SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD


echo "####################################################################"
echo "##  ${CONN_CONFIG} - VM: ListStatus"
echo "####################################################################"

curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX GET http://localhost:1024/spider/vmstatus -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
