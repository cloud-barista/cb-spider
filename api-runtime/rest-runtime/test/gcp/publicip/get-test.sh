RESTSERVER=localhost

#정상 동작
#생성 후 전달되는 Name으로 조회
curl -X GET http://$RESTSERVER:1024/publicip/gcppublicip1?connection_name=gcp-config01 |json_pp
