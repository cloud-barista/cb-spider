
echo "####################################################################"
echo "## Full Test Scripts for CB-Spider IID Working Version - 2020.04.22."
echo "##   1. VPC: Create -> Add-Subnet -> List -> Get"
echo "##   2. SecurityGroup: Create -> List -> Get"
echo "##   3. KeyPair: Create -> List -> Get"
echo "##   4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "## ---------------------------------"
echo "##   a. VPC: Register -> Unregister"
echo "##   b. SecurityGroup: Register -> Unregister"
echo "##   c. KeyPair: Register -> Unregister"
echo "##   d. VM: Register -> Unregister"
echo "## ---------------------------------"
echo "##   4. VM: Terminate(Delete)"
echo "##   3. KeyPair: Delete"
echo "##   2. SecurityGroup: Delete"
echo "##   1. VPC: Remove-Subnet -> Delete"
echo "####################################################################"

echo "####################################################################"
echo "## 1. VPC: Create -> Add-Subnet -> List -> Get"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spctl vpc create --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
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

$CBSPIDER_ROOT/interface/spctl vpc add-subnet --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "VPCName": "vpc-01", 
      "ReqInfo": { 
        "Name": "subnet-02", 
        "IPv4_CIDR": "'${IPv4_CIDR_SUBNET2}'"
      } 
    }'  

$CBSPIDER_ROOT/interface/spctl vpc list --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spctl vpc get --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vpc-01

echo "#-----------------------------"

echo "####################################################################"
echo "## 2. SecurityGroup: Create -> List -> Get"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spctl security create --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
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

$CBSPIDER_ROOT/interface/spctl security list --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spctl security get --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n sg-01

echo "#-----------------------------"

echo "####################################################################"
echo "## 3. KeyPair: Create -> List -> Get"
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

echo "#-----------------------------"

echo "####################################################################"
echo "## 4. VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spctl vm start --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
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

echo "============== sleep 1 after start VM"
sleep 1

$CBSPIDER_ROOT/interface/spctl vm list --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spctl vm get --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vm-01

$CBSPIDER_ROOT/interface/spctl vm liststatus --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}"

$CBSPIDER_ROOT/interface/spctl vm getstatus --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vm-01

$CBSPIDER_ROOT/interface/spctl vm suspend --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vm-01

echo "============== sleep 1 after suspend VM"
sleep 1

$CBSPIDER_ROOT/interface/spctl vm resume --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vm-01
echo "============== sleep 1 after resume VM"
sleep 1

$CBSPIDER_ROOT/interface/spctl vm reboot --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vm-01

echo "============== sleep 1 after reboot VM"
sleep 1
echo "#-----------------------------"


echo "####################################################################"
echo "####################################################################"
echo "####################################################################"

echo "####################################################################"
echo "## a. VPC: Register -> Unregister"
echo "####################################################################"


VPCINFO=$($CBSPIDER_ROOT/interface/spctl vpc get --config $CBSPIDER_ROOT/interface/spctl.conf -o json --cname "${CONN_CONFIG}" -n vpc-01)
VPCSystemId=$(jq '.IId.SystemId' <<<"$VPCINFO")
echo "$VPCSystemId" | jq ''

$CBSPIDER_ROOT/interface/spctl vpc register --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": { 
        "Name": "vpc-register-01", 
        "CSPId": '${VPCSystemId}'
      } 
    }' 

$CBSPIDER_ROOT/interface/spctl vpc unregister --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vpc-register-01

echo "#-----------------------------"

echo "####################################################################"
echo "## b. SecurityGroup: Register -> Unregister"
echo "####################################################################"


SECURITYINFO=$($CBSPIDER_ROOT/interface/spctl security get --config $CBSPIDER_ROOT/interface/spctl.conf -o json --cname "${CONN_CONFIG}" -n sg-01)
SecuritySystemId=$(jq '.IId.SystemId' <<<"$SECURITYINFO")
echo "$SecuritySystemId" | jq ''

$CBSPIDER_ROOT/interface/spctl security register --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": { 
        "VPCName": "vpc-01", 
        "Name": "sg-register-01", 
        "CSPId": '${SecuritySystemId}'
      } 
    }' 

$CBSPIDER_ROOT/interface/spctl security unregister --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n sg-register-01

echo "#-----------------------------"

echo "####################################################################"
echo "## c. KeyPair: Register -> Unregister"
echo "####################################################################"

KEYPAIRINFO=$($CBSPIDER_ROOT/interface/spctl keypair get --config $CBSPIDER_ROOT/interface/spctl.conf -o json --cname "${CONN_CONFIG}" -n keypair-01)
KeyPairSystemId=$(jq '.IId.SystemId' <<<"$KEYPAIRINFO")
echo "$KeyPairSystemId" | jq ''

$CBSPIDER_ROOT/interface/spctl keypair register --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": { 
        "Name": "keypair-register-01", 
        "CSPId": '${KeyPairSystemId}'
      } 
    }' 

$CBSPIDER_ROOT/interface/spctl keypair unregister --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n keypair-register-01

echo "#-----------------------------"

echo "####################################################################"
echo "## d. VM: Register -> Unregister"
echo "####################################################################"

VMINFO=$($CBSPIDER_ROOT/interface/spctl vm get --config $CBSPIDER_ROOT/interface/spctl.conf -o json --cname "${CONN_CONFIG}" -n vm-01)
VMSystemId=$(jq '.IId.SystemId' <<<"$VMINFO")
echo "$VMSystemId" | jq ''

$CBSPIDER_ROOT/interface/spctl vm register --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{ 
      "ConnectionName":"'${CONN_CONFIG}'",
      "ReqInfo": { 
        "Name": "vm-register-01", 
        "CSPId": '${VMSystemId}'
      } 
    }' 

$CBSPIDER_ROOT/interface/spctl vm unregister --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vm-register-01

echo "#-----------------------------"

echo "####################################################################"
echo "####################################################################"
echo "####################################################################"

echo "####################################################################"
echo "## 4. VM: Terminate(Delete)"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spctl vm terminate --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vm-01 --force false
echo "============== sleep 1 after delete VM"
sleep 1 

echo "####################################################################"
echo "## 3. KeyPair: Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spctl keypair delete --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n keypair-01 --force false
echo "####################################################################"
echo "## 2. SecurityGroup: Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spctl security delete --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n sg-01 --force false
echo "####################################################################"
echo "## 1. VPC: Remove-Subnet -> Delete"
echo "####################################################################"
$CBSPIDER_ROOT/interface/spctl vpc remove-subnet --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" --vname vpc-01 --sname subnet-02 --force false
$CBSPIDER_ROOT/interface/spctl vpc delete --config $CBSPIDER_ROOT/interface/spctl.conf --cname "${CONN_CONFIG}" -n vpc-01 --force false
