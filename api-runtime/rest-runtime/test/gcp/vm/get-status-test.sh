RESTSERVER=localhost

#정상 동작
#id로 조회
curl -X GET http://$RESTSERVER:1024/vmstatus/i-075d740aaaa410193?connection_name=gcp-config01 |json_pp

