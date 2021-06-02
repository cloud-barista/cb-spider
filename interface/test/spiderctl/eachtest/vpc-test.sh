
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version               "
echo "##   VPC: Create -> Add-Subnet -> List -> Get"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spctl vpc create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": { 
        "Name": "vpc-01", 
        "IPv4_CIDR": "'${IPv4_CIDR}'", 
        "SubnetInfoList": [ 
          { 
            "Name": "subnet-01", 
            "IPv4_CIDR": "'${IPv4_CIDR_SUBNET1}'"
          } 
        ] 
      } 
    }'

$CBSPIDER_ROOT/interface/spctl vpc add-subnet --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
'{ 
  "ConnectionName":"'${CONN_CONFIG}'",
  "VPCName": "vpc-01", 
  "ReqInfo": { 
    "Name": "subnet-02", 
    "IPv4_CIDR": "'${IPv4_CIDR_SUBNET2}'"
  } 
}' 

$CBSPIDER_ROOT/interface/spctl vpc list --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spctl vpc get --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n vpc-01


