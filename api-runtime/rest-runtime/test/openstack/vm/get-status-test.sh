RESTSERVER=localhost

#VM_ID=02c71331-558e-4ce7-b61a-cb8f8faaf270
VM_ID=vm-powerkim01
curl -X GET http://$RESTSERVER:1024/vmstatus/$VM_ID?connection_name=openstack-config01
