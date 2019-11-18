RESTSERVER=localhost

VM_ID=CBVm
curl -X GET http://$RESTSERVER:1024/vmstatus/$VM_ID?connection_name=openstack-config01
