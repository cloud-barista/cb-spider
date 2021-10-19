
echo "####################################################################"
echo "## KeyPair Test Scripts for CB-Spider IID Working Version - 2021.10.19."
echo "##   KeyPair: Register KeyPair"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/regkeypair -H 'Content-Type: application/json' -d \
       '{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": { "Name": "'${NAME}'", "CSPId": "'${CSPID}'"}
	}' |json_pp

