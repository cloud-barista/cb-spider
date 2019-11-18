RESTSERVER=localhost

VM_ID=CBVm
curl -X DELETE http://$RESTSERVER:1024/vm/$VM_ID?connection_name=openstack-config01
