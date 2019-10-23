RESTSERVER=192.168.130.8

#vNetWorkId의 경우 생성시 자동 할당
vNetworkId=43dcec05-a3a4-47dc-a342-1f673cb3f39d

curl -X DELETE http://$RESTSERVER:1024/vnetwork/$vNetworkId?connection_name=openstack-config01