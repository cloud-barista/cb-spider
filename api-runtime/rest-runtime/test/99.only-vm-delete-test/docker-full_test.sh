
echo "####################################################################"
echo "##   4. VM: Terminate(Delete)"
echo "##   3. KeyPair: DOCKER does noet support."
echo "##   2. SecurityGroup: DOCKER does noet support."
echo "##   1. VPC: DOCKER does noet support."
echo "####################################################################"

echo "####################################################################"
echo "## 4. VM: Terminate(Delete)"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vm/${CONN_CONFIG}-vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "============== sleep 0 after delete VM"
sleep 0 

echo "####################################################################"
echo "## 3. KeyPair: DOCKER does noet support."
echo "####################################################################"
echo "####################################################################"
echo "## 2. SecurityGroup: DOCKER does noet support."
echo "####################################################################"
echo "####################################################################"
echo "## 1. VPC: DOCKER does noet support."
echo "####################################################################"

