RESTSERVER=node12

SECURITY_NAME=CB-SecGroup
curl -X DELETE http://$RESTSERVER:1024/securitygroup/$SECURITY_NAME?connection_name=azure-config01
