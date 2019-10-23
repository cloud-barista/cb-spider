RESTSERVER=127.0.0.1

curl -X POST http://$RESTSERVER:1024/securitygroup?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{
"Name": "CB-SecGroup",
"SecurityRules": [
{"FromPort": "22"},
{"ToPort" : "22"},
{"IPProtocol" : "TCP"},
{"Direction" : "inbound"} ] }'
