RESTSERVER=localhost

#정상 동작
curl -X DELETE http://$RESTSERVER:1024/securitygroup/security01?connection_name=gcp-config01
