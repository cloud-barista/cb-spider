RESTSERVER=localhost

# 서버 기능 미지원 - {"message":" is not a valid action!!"} 에러 발생
curl -X GET http://$RESTSERVER:1024/controlvm/i-075d740aaaa410193?connection_name=aws-config01&action=resume |json_pp
