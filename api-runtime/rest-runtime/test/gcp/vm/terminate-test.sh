RESTSERVER=localhost

#서버 에러 발생 - "message" : "Method Not Allowed"
curl -X DELETE http://$RESTSERVER:1024/vm/cscmcloud?connection_name=gcp-config01 |json_pp
