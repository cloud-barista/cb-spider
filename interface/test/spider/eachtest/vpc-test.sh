
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version               "
echo "##   VPC: Create -> List -> Get"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spider vpc create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": { 
        "Name": "vpc-01", 
        "IPv4_CIDR": "'${IPv4_CIDR}'", 
        "SubnetInfoList": [ 
          { 
            "Name": "subnet-01", 
            "IPv4_CIDR": "'${IPv4_CIDR}'"
          } 
        ] 
      } 
    }'

$CBSPIDER_ROOT/interface/spider vpc list --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spider vpc get --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vpc-01


