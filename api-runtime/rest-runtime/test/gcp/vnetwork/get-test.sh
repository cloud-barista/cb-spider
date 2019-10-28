RESTSERVER=localhost

#정상 동작
#ID로 조회해야 함.

#[참고]
#ID가 전달되면 ID로 조회하고 ID를 전달하지 않으면 Tag중 "CB-VNet-Subnet" 값으로 내부에서 자체 조회 함.
curl -X GET http://$RESTSERVER:1024/vnetwork/subnet-04c4aae5c8d08f64d?connection_name=aws-config01 |json_pp
