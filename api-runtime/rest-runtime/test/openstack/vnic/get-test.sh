RESTSERVER=192.168.130.8

#vNicId의 경우 생성시 자동 할당
vNicId= d8c7e8ed-5981-4568-8327-7451472c69f2

# name -> ID로 변경
curl -X GET http://$RESTSERVER:1024/vnic/$vNicId?connection_name=openstack-config01
