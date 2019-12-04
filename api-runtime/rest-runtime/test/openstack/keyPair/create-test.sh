RESTSERVER=localhost

# CB-KeyPair 이름으로 키페어 생성 및 반환 처리

curl -X POST http://$RESTSERVER:1024/keypair?connection_name=openstack-config01 -H 'Content-Type: application/json' -d '{ "Name": "CB-Keypair" }' |json_pp
