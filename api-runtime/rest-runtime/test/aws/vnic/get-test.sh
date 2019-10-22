RESTSERVER=localhost

#정상 동작
#id로 조회
curl -X GET http://$RESTSERVER:1024/vnic/eni-085ef6187e4b1a54a?connection_name=aws-config01 |json_pp
