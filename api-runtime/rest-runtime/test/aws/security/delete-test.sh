RESTSERVER=localhost

#정상 동작
#id로 삭제
curl -X DELETE http://$RESTSERVER:1024/securitygroup/sg-08799d5f21ada740c?connection_name=aws-config01
