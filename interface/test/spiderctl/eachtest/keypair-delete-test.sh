
echo "####################################################################"
echo "## KeyPair Test Scripts for CB-Spider IID Working Version           "
echo "##   KeyPair: Delete"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spctl keypair delete --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n keypair-01 --force false
