
echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   Create: VPC/Subnet -> SG -> Key -> vm-01 -> vm-02 "
echo "####################################################################"

echo ""

echo "#####---------- CreateVPC ----------####"
curl -sX POST http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { 
			"Name": "vpc-01", 
			"IPv4_CIDR": "10.0.0.0/16", 
			"SubnetInfoList": [ { "Name": "subnet-01", "IPv4_CIDR": "10.0.8.0/22"} ]
			} 
	}' |json_pp

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi


echo "#####---------- CreateSG ----------####"
curl -sX POST http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { 
			"Name": "sg-01", 
			"VPCName": "vpc-01", 
			"SecurityRules": [ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound"}, {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "udp", "Direction" : "inbound"} ] 
		} 
	}' |json_pp

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

CLIPATH=$CBSPIDER_ROOT/interface
KEYPAIR_NAME=$1-keypair-01

echo "#####---------- CreateKey ----------####"
curl -sX POST http://localhost:1024/spider/keypair -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { "Name": "'${KEYPAIR_NAME}'" } 
	}' |json_pp

ret=`$CLIPATH/spctl --config $CLIPATH/spctl.conf keypair create -i json -o json -d \
    '{
		"ConnectionName":"'${CONN_CONFIG}'",
		"ReqInfo": {
        "Name": "'${KEYPAIR_NAME}'"
		}
    }'`

echo -e "$ret"

result=`echo -e "$ret" | grep already`
if [ "$result" ];then
        echo "You already have the private Key."
else
        echo "$ret" | grep PrivateKey | sed 's/  "PrivateKey": "//g' | sed 's/",//g' | sed 's/\\n/\n/g' > ${KEYPAIR_NAME}.pem
        chmod 600 ${KEYPAIR_NAME}.pem
fi

echo -e "\n\n"

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi


echo "#####---------- StartVM:vm-01 ----------####"
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { 
			"Name": "vm-01", 
			"ImageName": "'${IMAGE_NAME}'", 
			"VMSpecName": "'${SPEC_NAME}'", 
			"VPCName": "vpc-01", 
			"SubnetName": "subnet-01", 
			"SecurityGroupNames": [ "sg-01" ], 
			"KeyPairName": "'${KEYPAIR_NAME}'"
		} 
	}' |json_pp

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

echo "#####---------- StartVM:vm-02 ----------####"
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Name": "vm-02",
                        "ImageName": "'${IMAGE_NAME}'",
                        "VMSpecName": "'${SPEC_NAME}'",
                        "VPCName": "vpc-01",
                        "SubnetName": "subnet-01",
                        "SecurityGroupNames": [ "sg-01" ],
                        "KeyPairName": "'${KEYPAIR_NAME}'"
                }
        }' |json_pp

