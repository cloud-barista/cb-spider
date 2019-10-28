RESTSERVER=localhost

# 서버 기능 미지원 - {"message":" is not a valid action!!"} 에러 발생
curl -X GET "http://$RESTSERVER:1024/controlvm/i-03665b7acbb9f623a?connection_name=aws-config01&action=resume"
