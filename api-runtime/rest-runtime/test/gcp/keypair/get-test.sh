RESTSERVER=localhost

#정상 동작
#Name으로 조회
curl -X GET http://$RESTSERVER:1024/spider/keypair/mcb-keypair?connection_name=gcp-config01 |json_pp