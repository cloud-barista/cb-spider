RESTSERVER=localhost

# Azure에서 별도로 KeyPair 기능을 제공하지 않기 때문에 내부적으로 keypair 생성 후 해당 keypair 반환
# CB-KeyPair 이름으로 키페어 생성 및 반환 처리

curl -X POST http://$RESTSERVER:1024/keypair?connection_name=azure-config01 -H 'Content-Type: application/json' -d '{ "Name": "CB-KeyPair" }' |json_pp
