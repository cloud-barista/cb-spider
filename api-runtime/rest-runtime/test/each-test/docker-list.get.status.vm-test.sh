
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   VM: List -> Get -> ListStatus -> GetStatus "
echo "####################################################################"

curl -sX GET http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vm/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vmstatus -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vmstatus/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

