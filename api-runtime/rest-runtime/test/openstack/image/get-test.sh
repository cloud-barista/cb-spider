RESTSERVER=localhost

# OpenStack의 경우 사용자가 등록한 이미지 정보를 조회
# ubuntu16.04 이름을 기준으로 이미지 정보 조회

IMAGE_ID=ubuntu16.04
curl -X GET http://$RESTSERVER:1024/vmimage/$IMAGE_ID?connection_name=openstack-config01 |json_pp
