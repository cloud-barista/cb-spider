RESTSERVER=localhost

#정상 동작
#name으로 조회
curl -X GET http://$RESTSERVER:1024/vm/vm01?connection_name=gcp-config01 |json_pp
