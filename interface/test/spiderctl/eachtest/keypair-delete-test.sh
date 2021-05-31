
echo "####################################################################"
echo "## KeyPair Test Scripts for CB-Spider IID Working Version           "
echo "##   KeyPair: Delete"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spiderctl keypair delete --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n keypair-01 --force false
