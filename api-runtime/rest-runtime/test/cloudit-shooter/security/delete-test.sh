source ../setup.env

SECURITY_ID=2f8a0400-3740-47ea-84bc-d661ca8135fa
curl -X DELETE http://$RESTSERVER:1024/securitygroup/$SECURITY_ID?connection_name=cloudit-config01 | json_pp
