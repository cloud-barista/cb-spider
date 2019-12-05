RESTSERVER=localhost

#정상 동작
# Vnetwork 를 지정해 줘야 해당 vnet에 해당하는 VM들한테 정책을 할당할 수 있다.
#인바운드 정책만 존재하는 경우 테스트
curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=gcp-config01 -H 'Content-Type: application/json' -d '{ "Name": "security01", "SecurityRules": [ {"FromPort": "20", "ToPort" : "22", "IPProtocol" : "tcp", "Direction" : "INGRESS"} ] }' |json_pp
