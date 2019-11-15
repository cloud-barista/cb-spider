RESTSERVER=localhost

VM_ID=vm-powerkim01
curl -X DELETE http://$RESTSERVER:1024/vm/$VM_ID?connection_name=openstack-config01
