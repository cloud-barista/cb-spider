
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version - 2021.10.18."
echo "##   VPC: Unregister VPC"
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/regvpc/${VPC_NAME} -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'" 
	}' |json_pp
