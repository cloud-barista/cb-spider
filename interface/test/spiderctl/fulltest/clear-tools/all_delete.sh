
echo "####################################################################"
echo "## Full Test Scripts for CB-Spider IID Working Version              "
echo "##   4. VM: Terminate(Delete)"
echo "##   3. KeyPair: Delete"
echo "##   2. SecurityGroup: Delete"
echo "##   1. VPC: Delete"
echo "####################################################################"

echo "####################################################################"
echo "## 4. VM: Terminate(Delete)"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spiderctl vm terminate --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vm-01 --force false
echo "============== sleep 15 after delete VM"
sleep 15 

echo "####################################################################"
echo "## 3. KeyPair: Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spiderctl keypair delete --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n keypair-01 --force false

echo "####################################################################"
echo "## 2. SecurityGroup: Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spiderctl security delete --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n sg-01 --force false

echo "####################################################################"
echo "## 1. VPC: Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spiderctl vpc delete --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vpc-01 --force false


