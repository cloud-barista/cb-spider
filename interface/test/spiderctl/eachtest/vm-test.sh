
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version                "
echo "##   VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spctl vm start --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": { 
        "Name": "vm-01", 
        "ImageName": "'${IMAGE_NAME}'", 
        "VPCName": "vpc-01", 
        "SubnetName": "subnet-01", 
        "SecurityGroupNames": [ "sg-01" ], 
        "VMSpecName": "'${SPEC_NAME}'", 
        "KeyPairName": "keypair-01"
      } 
    }'

echo "============== sleep 60 after start VM"
sleep 60

$CBSPIDER_ROOT/interface/spctl vm list --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spctl vm get --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vm-01

$CBSPIDER_ROOT/interface/spctl vm liststatus --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spctl vm getstatus --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vm-01

$CBSPIDER_ROOT/interface/spctl vm suspend --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vm-01

echo "============== sleep 60 after suspend VM"
sleep 60

$CBSPIDER_ROOT/interface/spctl vm resume --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vm-01

echo "============== sleep 30 after resume VM"
sleep 30

$CBSPIDER_ROOT/interface/spctl vm reboot --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vm-01

echo "============== sleep 60 after reboot VM"
sleep 60 


