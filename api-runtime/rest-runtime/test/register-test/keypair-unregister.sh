
echo "####################################################################"
echo "## KeyPair Test Scripts for CB-Spider IID Working Version - 2021.10.19."
echo "##   KeyPair: Unregister KeyPair"
echo "####################################################################"

curl -sX DELETE http://localhost:1024/spider/regkeypair/${NAME} -H 'Content-Type: application/json' -d \
       '{
		"ConnectionName": "'${CONN_CONFIG}'" 
	}' |json_pp

