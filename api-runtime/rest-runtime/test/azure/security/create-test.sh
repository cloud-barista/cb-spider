RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{ "Name": "CB-SecGroup", "SecurityRules": [{"FromPort": "0", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound"}] }'
