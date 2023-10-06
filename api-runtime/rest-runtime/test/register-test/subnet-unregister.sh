
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version - 2023.10.06."
echo "##   VPC: Unregister Subnet"
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/regsubnet/${Subnet_NAME} -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'" ,
                "ReqInfo": { "VPCName": "'${VPC_NAME}'"}
	}' |json_pp
