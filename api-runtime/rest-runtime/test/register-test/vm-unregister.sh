
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2021.10.19."
echo "##   VM: Unregister VM"
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/regvm/${NAME} -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'"
	}' |json_pp

