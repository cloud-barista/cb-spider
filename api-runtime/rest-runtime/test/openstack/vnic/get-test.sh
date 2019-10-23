RESTSERVER=localhost

VNIC_ID=e52b5c20-bb92-4af7-8caa-a9b317d647f9
curl -X GET http://$RESTSERVER:1024/vnic/$VNIC_ID?connection_name=openstack-config01
