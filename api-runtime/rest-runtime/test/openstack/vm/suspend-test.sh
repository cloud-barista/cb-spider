RESTSERVER=localhost

VM_ID=CBVm
curl -X GET "http://$RESTSERVER:1024/controlvm/$VM_ID?connection_name=openstack-config01&action=suspend"
