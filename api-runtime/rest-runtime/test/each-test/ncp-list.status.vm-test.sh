
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2020.10.20."
echo "##   VM: ListStatus"
echo "####################################################################"

curl -sX GET http://localhost:1024/spider/vmstatus -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
