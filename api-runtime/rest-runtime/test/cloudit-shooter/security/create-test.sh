source ../setup.env

curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{ "Name": "security01-powerkim", "SecurityRules": [{"FromPort": "22", "ToPort" : "22", "IPProtocol" : "tcp", "Direction" : "inbound"}] }' |json_pp
