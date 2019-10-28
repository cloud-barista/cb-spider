RESTSERVER=localhost

# 의존성 에러 발생 - Subnet 삭제 후 VPC 제거 시 새로 추가된 리소스들로 인한 의존성이 발생함.
# 의존성 에러 처리 - VPC에 새로 추가한 라우팅 테이블및 IGW의 라우터를 제거해야 삭제 가능함. (자동 제거 로직 추가 예정)

#ID로 삭제해야 함.

#[참고]
# 삭제되는 시점에 따라서 일부 리소스의 의존성 때문에 삭제가 안되는 경우가 있음.
curl -X DELETE http://$RESTSERVER:1024/vnetwork/subnet-04c4aae5c8d08f64d?connection_name=gcp-config01 |json_pp
