
echo "####################################################################"
echo "## Full Test Scripts for CB-Spider IID Working Version - 2020.04.22."
echo "##   1. VPC: DOCKER does noet support."
echo "##   2. SecurityGroup: DOCKER does noet support."
echo "##   3. KeyPair:  DOCKER does noet support."
echo "##   4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"

echo "####################################################################"
echo "## 1. VPC: DOCKER does noet support."
echo "####################################################################"
echo "#-----------------------------"

echo "####################################################################"
echo "## 2. SecurityGroup: DOCKER does noet support."
echo "####################################################################"
echo "#-----------------------------"

echo "####################################################################"
echo "## 3. KeyPair: DOCKER does noet support."
echo "####################################################################"
echo "#-----------------------------"

echo "####################################################################"
echo "## 4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "'${CONN_CONFIG}'-vm-01", "ImageName": "'${IMAGE_NAME}'" } }' |json_pp
