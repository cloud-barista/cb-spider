
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version               "
echo "##   VPC: Remove-Subnet -> Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spider vpc remove-subnet --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" --vname vpc-01 --sname subnet-02 --force false
$CBSPIDER_ROOT/interface/spider vpc delete --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vpc-01 --force false


