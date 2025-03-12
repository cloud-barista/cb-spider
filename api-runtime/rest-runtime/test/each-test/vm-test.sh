
echo "####################################################################"
echo "## VM Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   VM: StartVM -> List -> Get -> ListStatus -> GetStatus -> Suspend -> Resume -> Reboot"
echo "####################################################################"

curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d "{ \"ConnectionName\": \"${CONN_CONFIG}\", \"ReqInfo\": { \"Name\": \"vm-01\", \"ImageName\": \"${IMAGE_NAME}\", \"VPCName\": \"vpc-01\", \"SubnetName\": \"subnet-01\", \"SecurityGroupNames\": [ \"sg-01\" ], \"VMSpecName\": \"${SPEC_NAME}\", \"KeyPairName\": \"keypair-01\"} }" |json_pp

echo "============== sleep 30 after start VM"
sleep 30
curl -sX GET http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vm/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vmstatus -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/vmstatus/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
curl -sX GET http://localhost:1024/spider/controlvm/vm-01?action=suspend -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "============== sleep 50 after suspend VM"
sleep 50
curl -sX GET http://localhost:1024/spider/controlvm/vm-01?action=resume -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "============== sleep 30 after resume VM"
sleep 30
curl -sX GET http://localhost:1024/spider/controlvm/vm-01?action=reboot -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "============== sleep 60 after reboot VM"
sleep 60 

