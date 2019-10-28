RESTSERVER=localhost

#정상 동작
#id로 조회
curl -X GET http://$RESTSERVER:1024/vm/i-075d740aaaa410193?connection_name=aws-config01 |json_pp
