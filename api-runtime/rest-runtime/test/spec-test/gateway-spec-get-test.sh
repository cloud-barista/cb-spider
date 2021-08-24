
echo "####################################################################"
echo "## Spec Test Scripts for CB-Spider IID Working Version - 2020.12.03."
echo "##   Spec: Get"
echo "####################################################################"

time curl -sX GET http://${GATEWAY_HOST}:8000/spider/vmspec/${SPEC_NAME} -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
