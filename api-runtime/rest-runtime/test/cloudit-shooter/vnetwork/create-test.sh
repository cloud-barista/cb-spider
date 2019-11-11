source ../setup.env

curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{"Name":"CB-VNet-powerkim"}'| json_pp
