
echo "####################################################################"
echo "## Image Test Scripts for CB-Spider IID Working Version - 2020.12.03."
echo "##   Image: Get"
echo "####################################################################"

time curl -sX GET http://${GATEWAY_HOST}:8000/spider/vmimage/${IMAGE_NAME} -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
