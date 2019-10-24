RESTSERVER=localhost

# [참고]
# Cloudit에서 PublicIP 생성 시 VM의 Private IP 주소 값이 필요
# 현재 KeyValueList Map에 PrivateIP 항목으로 매핑해서 전달
# 예시) 10.0.0.2

curl -X POST http://$RESTSERVER:1024/publicip?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{ "Name": "CB-PublicIP", "KeyValueList": [{"Key":"PrivateIP", "Value":"10.0.0.2"}]}'
