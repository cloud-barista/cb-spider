RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/spider/securitygroup?connection_name=cloudit-config01 | json_pp
