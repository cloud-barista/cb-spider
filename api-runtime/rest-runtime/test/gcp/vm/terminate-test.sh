RESTSERVER=localhost

#완료
curl -X DELETE http://$RESTSERVER:1024/spider/vm/vm01?connection_name=gcp-config01 |json_pp
