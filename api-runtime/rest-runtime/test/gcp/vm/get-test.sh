RESTSERVER=localhost

#정상 동작
curl -X GET http://$RESTSERVER:1024/spider/vm/vm01?connection_name=gcp-config01 |json_pp
