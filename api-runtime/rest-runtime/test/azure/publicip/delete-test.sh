RESTSERVER=localhost

PUBLICIP_NAME=CB-PublicIP
curl -X DELETE http://$RESTSERVER:1024/spider/publicip/$PUBLICIP_NAME?connection_name=azure-config01
