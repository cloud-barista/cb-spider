RESTSERVER=192.168.130.8

#생성시 Pool내의 IP자동할당
publicIPId = 182.252.135.157

curl -X DELETE http://$RESTSERVER:1024/publicip/$publicIPId?connection_name=openstack-config01
