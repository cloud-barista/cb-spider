RESTSERVER=localhost

#정상 동작
#id로 삭제
curl -X DELETE http://$RESTSERVER:1024/spider/vnic/eni-085ef6187e4b1a54a?connection_name=gcp-config01 |json_pp
