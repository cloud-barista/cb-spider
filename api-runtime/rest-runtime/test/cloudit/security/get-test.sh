RESTSERVER=localhost

SECURITY_ID=6a3bcb31-c172-43ba-a7d0-4ae17dcf74bd
curl -X GET http://$RESTSERVER:1024/securitygroup/$SECURITY_ID?connection_name=cloudit-config01 | json_pp
