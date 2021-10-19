
echo "####################################################################"
echo "## SecurityGroup Test Scripts for CB-Spider IID Working Version - 2021.10.18."
echo "##   SecurityGroup: Register SecurityGroup"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/regsecuritygroup -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { "VPCName": "'${VPC_NAME}'", "Name": "'${SG_NAME}'", "CSPId": "'${SG_CSPID}'"} 
	}' |json_pp
