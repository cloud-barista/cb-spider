
echo "####################################################################"
echo "## SecurityGroup Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   SecurityGroup: Create -> List -> Get"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spider security create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": { 
        "Name": "sg-01", 
        "VPCName": "vpc-01", 
        "SecurityRules": [ 
          {
            "FromPort": "1", 
            "ToPort" : "65535", 
            "IPProtocol" : "tcp", 
            "Direction" : "inbound",
            "CIDR" : "0.0.0.0/0"
          }
        ] 
      } 
    }'

$CBSPIDER_ROOT/interface/spider security list --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spider security get --config $CBSPIDER_ROOT/interface/grpc_conf.yaml --cname "${CONN_CONFIG}" -n sg-01
       
 