
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version               "
echo "##   VPC: Delete"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spider vpc delete --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vpc-01


