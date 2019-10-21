RESTSERVER=node12

curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{"Name":"cb-vnet-subnet01"}'
