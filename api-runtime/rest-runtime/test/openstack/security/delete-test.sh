RESTSERVER=192.168.130.8

#securityGroupId 경우 생성시 자동 할당
securityGroupId = c2b6e1d6-dbb1-4100-818c-0f75cc89470c

# subnetName -> ID로 변경 해야됨
curl -X DELETE http://$RESTSERVER:1024/securitygroup/$securityGroupId?connection_name=openstack-config01
