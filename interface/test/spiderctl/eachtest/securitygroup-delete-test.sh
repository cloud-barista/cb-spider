
echo "####################################################################"
echo "## SecurityGroup Test Scripts for CB-Spider IID Working Version     "
echo "##   SecurityGroup: Delete"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spctl security delete --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n sg-01  --force false
