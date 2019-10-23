RESTSERVER=localhost

# [참고]
# 기본 네트워크인 CB-VNet 하위에 서브넷 생성
# 서브넷 삭제 시 자동으로 라우터 인터페이스 삭제

VNETWORK_ID=93783b70-92ec-47ce-9739-2bdc6df614eb
curl -X DELETE http://$RESTSERVER:1024/vnetwork/$VNETWORK_ID?connection_name=openstack-config01
