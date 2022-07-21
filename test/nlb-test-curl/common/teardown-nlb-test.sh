
echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   Create: VPC/Subnet -> SG -> Key -> vm-01 -> vm-02 "
echo "####################################################################"

echo ""


echo "#####---------- TerminateVM:vm-01 ----------####"
curl -sX DELETE http://localhost:1024/spider/vm/vm-01 -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

echo "#####---------- TerminateVM:vm-02 ----------####"
curl -sX DELETE http://localhost:1024/spider/vm/vm-02 -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp


if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

echo "#####---------- DeleteKey ----------####"

KEYPAIR_NAME=$1-keypair-01

curl -sX DELETE http://localhost:1024/spider/keypair/$KEYPAIR_NAME -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp

rm -f ./${KEYPAIR_NAME}.pem

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi


echo "#####---------- DeleteSG ----------####"
curl -sX DELETE http://localhost:1024/spider/securitygroup/sg-01 -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp


if [ "$SLEEP" ]; then
        sleep $SLEEP
fi


echo "#####---------- DeleteVPC ----------####"
curl -sX DELETE http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'"
	}' |json_pp

