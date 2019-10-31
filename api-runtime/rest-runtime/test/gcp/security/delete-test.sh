RESTSERVER=localhost

#정상 동작
#id로 삭제
curl -X DELETE http://$RESTSERVER:1024/securitygroup/security01?connection_name=gcp-config01
