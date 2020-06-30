
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
$CBSPIDER_ROOT/interface/spider vm terminate --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vm-01
echo "============== sleep 15 after delete VM"
sleep 15 

echo "####################################################################"
echo "## 3. KeyPair: Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spider keypair delete --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n keypair-01

echo "####################################################################"
echo "## 2. SecurityGroup: Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spider security delete --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n sg-01

echo "####################################################################"
echo "## 1. VPC: Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spider vpc delete --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vpc-01


