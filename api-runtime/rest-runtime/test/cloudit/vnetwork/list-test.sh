RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/vnetwork?connection_name=cloudit-config01 | json_pp
