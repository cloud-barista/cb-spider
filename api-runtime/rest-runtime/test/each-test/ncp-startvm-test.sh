
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2020.09.15."
echo "##   VM: StartVM "
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "'${CONN_CONFIG}'-vm-001", "ImageName": "'${IMAGE_NAME}'" } }' |json_pp

# curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "'${CONN_CONFIG}'-vm-01", "ImageName": "'${IMAGE_NAME}'", "VMSpecName": "'${SPEC_NAME}'", "KeyPairName": "'${KEYPAIR_NAME}'" } }' |json_pp

