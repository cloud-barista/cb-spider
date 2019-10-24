RESTSERVER=192.168.130.8

curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{
"Name": "mcb-test-sg",
"SecurityRules":
[{"FromPort": "22"},
{"ToPort" : "22"},
{"IPProtocol" : "TCP"},
{"Direction" : "inbound"}
] }'
