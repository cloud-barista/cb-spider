RESTSERVER=localhost

PUBLICIP_NAME=CB-PublicIP
curl -X GET http://$RESTSERVER:1024/publicip/$PUBLICIP_NAME?connection_name=azure-config01 |json_pp
