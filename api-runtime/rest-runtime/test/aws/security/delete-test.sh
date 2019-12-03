RESTSERVER=localhost

#정상 동작
curl -X DELETE http://$RESTSERVER:1024/securitygroup/cbsg01-in?connection_name=aws-config01
