
echo "####################################################################"
echo "## KeyPair Test Scripts for CB-Spider IID Working Version           "
echo "##   KeyPair: Create -> List -> Get"
echo "####################################################################"

$CBSPIDER_ROOT/interface/spctl keypair create --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": { 
        "Name": "keypair-01" 
      } 
    }'

$CBSPIDER_ROOT/interface/spctl keypair list --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spctl keypair get --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n keypair-01
    
