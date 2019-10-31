RESTSERVER=localhost

# 서버 기능 미지원 - {"message":" is not a valid action!!"} 에러 발생
curl -X GET "http://$RESTSERVER:1024/controlvm/vm01?connection_name=gcp-config01&action=resume"
