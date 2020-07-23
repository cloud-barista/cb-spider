
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version                "
echo "##   VM: Terminate"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spider vm terminate --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vm-01  --force false

echo "============== sleep 15 after delete VM"
sleep 15 

