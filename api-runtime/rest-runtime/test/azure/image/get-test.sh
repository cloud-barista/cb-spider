RESTSERVER=localhost

# Azure에서 제공하는 마켓플레이스 퍼블릭 이미지의 경우 URN 형태의 ID 제공 (Publisher:Offer:SKU:Version)
# URN 형태의 Image ID를 기준으로 이미지 정보 조회

IMAGE_ID=Canonical:UbuntuServer:18.04-LTS:latest
curl -X GET http://$RESTSERVER:1024/vmimage/$IMAGE_ID?connection_name=azure-config01 |json_pp
