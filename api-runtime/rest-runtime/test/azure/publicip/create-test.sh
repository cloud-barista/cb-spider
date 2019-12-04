RESTSERVER=localhost

# CB-PublicIP 이름으로 퍼블릭 IP 생성 및 반환 처리

curl -X POST http://$RESTSERVER:1024/publicip?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{ "Name": "CB-PublicIP" }' |json_pp
