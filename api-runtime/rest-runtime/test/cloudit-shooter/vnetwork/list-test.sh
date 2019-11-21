source ../setup.env

curl -sX GET http://$RESTSERVER:1024/vnetwork?connection_name=cloudit-config01 | json_pp
