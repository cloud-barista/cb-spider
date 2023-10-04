
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version - 2023.10.04."
echo "##   VPC: Register Subnet"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/regsubnet -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { "VPCName": "'${VPC_NAME}'", "Name": "'${Subnet_NAME}'", "CSPId": "'${Subnet_CSPID}'"} 
	}' |json_pp

