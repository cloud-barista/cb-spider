RESTSERVER=localhost

#정상 동작
#생성 후 전달되는 Name으로 삭제
curl -X DELETE http://$RESTSERVER:1024/publicip/gcppublicip1?connection_name=gcp-config01
