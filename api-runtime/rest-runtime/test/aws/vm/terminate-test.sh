RESTSERVER=localhost

#서버 에러 발생 - "message" : "Method Not Allowed"
curl -X DELETE http://$RESTSERVER:1024/vm/i-075d740aaaa410193?connection_name=aws-config01 |json_pp
