RESTSERVER=localhost

#정상 동작

#인바운드 정책만 존재하는 경우 테스트
curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{ "Name": "cbsg01-in", "SecurityRules": [ {"FromPort": "20", "ToPort" : "22", "IPProtocol" : "tcp", "Direction" : "inbound"} ] }' |json_pp

#아웃바운드 정책만 존재하는 경우 테스트
#curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{ "Name": "cbsg01-out", "SecurityRules": [{"FromPort": "22", "ToPort" : "22", "IPProtocol" : "tcp", "Direction" : "outbound"} ] }' |json_pp

#인바운드 / 아웃바운드 둘 다 존재하는 경우 테스트
#curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{ "Name": "cbsg01-inout", "SecurityRules": [ {"FromPort": "22", "ToPort" : "22", "IPProtocol" : "tcp", "Direction" : "inbound"},{"FromPort": "22", "ToPort" : "22", "IPProtocol" : "tcp", "Direction" : "outbound"} ] }' |json_pp
