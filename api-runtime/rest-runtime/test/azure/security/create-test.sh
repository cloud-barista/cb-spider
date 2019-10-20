RESTSERVER=node12

curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{ "Name": "security01", "SecurityRules": [ {"FromPort": "22"}, {"ToPort" : "22"}, {"IPProtocol" : "tcp"}, {"Direction" : "inbound"} ] }'
