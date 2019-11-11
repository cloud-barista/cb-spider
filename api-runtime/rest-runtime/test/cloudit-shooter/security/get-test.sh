source ../setup.env

#curl -X GET http://$RESTSERVER:1024/securitygroup/security01-powerkim?connection_name=cloudit-config01 |json_pp

SECURITY_ID=2f8a0400-3740-47ea-84bc-d661ca8135fa
curl -X GET http://$RESTSERVER:1024/securitygroup/$SECURITY_ID?connection_name=cloudit-config01 | json_pp
