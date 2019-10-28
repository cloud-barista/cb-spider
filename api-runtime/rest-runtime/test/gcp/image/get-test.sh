RESTSERVER=localhost

# 서버에서 전달 받은 값을 이용해서 Driver를 호출하지 않는 것 같음 - 현재는 서버가 무조건 404 Not found를 리턴 중
# AMI ID로 조회
#curl -X GET http://$RESTSERVER:1024/vmimage/ami-047f7b46bd6dd5d84/connection_name=aws-config01 |json_pp
curl -X GET http://$RESTSERVER:1024/vmimage/ami-047f7b46bd6dd5d84?connection_name=aws-config01 |json_pp
