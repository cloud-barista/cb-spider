RESTSERVER=localhost

#정상 동작
#id로 조회
curl -X GET http://$RESTSERVER:1024/securitygroup/sg-05178dba7d1ef476f?connection_name=gcp-config01 |json_pp
