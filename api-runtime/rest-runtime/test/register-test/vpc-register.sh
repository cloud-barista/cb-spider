
echo "####################################################################"
echo "## VPC Test Scripts for CB-Spider IID Working Version - 2021.10.18."
echo "##   VPC: Register VPC"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/regvpc -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { "Name": "'${VPC_NAME}'", "CSPId": "'${VPC_CSPID}'"} 
	}' |json_pp

