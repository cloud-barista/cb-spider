RESTSERVER=localhost

#정상 동작
#생성후 전달 받은 Name(AllocateID) 필드의 값을 이용해서 조회 및 삭제 가능 함.
curl -X POST http://$RESTSERVER:1024/publicip?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{ "Name": "cbpublicip01" }' |json_pp
