RESTSERVER=localhost

VM_NAME=CBVm
curl -X GET http://$RESTSERVER:1024/vmstatus/$VM_NAME?connection_name=openstack-config01
