
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2021.10.19."
echo "##   VM: Register VM"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/regvm -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { "Name": "'${NAME}'", "CSPId": "'${CSPID}'"}
	}' |json_pp

