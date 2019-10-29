RESTSERVER=localhost

#정상 동작
#Name으로 삭제
curl -X DELETE http://$RESTSERVER:1024/keypair/mcb-keypair?connection_name=gcp-config01 |json_pp