RESTSERVER=localhost

# CB-SecGroup 이름으로 보안그룹 생성 및 반환 처리

curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{
 "Name": "CB-SecGroup",
 "SecurityRules": [
    {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound"},
    {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "outbound"}
    ]
}' |json_pp
