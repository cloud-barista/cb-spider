
echo "####################################################################"
echo "## SecurityGroup Test Scripts for CB-Spider IID Working Version - 2021.10.18."
echo "##   SecurityGroup: Unregister SG"
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/regsecuritygroup/${SG_NAME} -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'"
	}' |json_pp

