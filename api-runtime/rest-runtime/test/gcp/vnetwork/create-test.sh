RESTSERVER=localhost

#정상 동작

# [동작]
# 내부에서 VPC를 자동으로 생서하고 Subnet도 자동으로 생성함.

# [참고]
# 자동 생성 기능으로 인해 1개의 VPC & Subnet만 생성 가능한데 생성할 Name을 전달받으면...
# VPC or Subnet 정보를 전달 받지 않고 로직을 처리해야하는 핸들러의 경우 자동으로 생성된 Subnet의 이름을 알 수 없기 때문에
# 고정된 이름을 이용해서 자동으로 생성된 서브넷 값을 찾기 위해 전달받은 Name 값은 무시함.
# 따라서, 사용자가 전달하는 Name값과 무관하게 "CB-VNet-Subnet" 명칭으로 생성됨.
curl -X POST http://$RESTSERVER:1024/vnetwork?connection_name=aws-config01 -H 'Content-Type: application/json' -d '{"Name":"CB-VNet-Subnet"}' |json_pp
