RESTSERVER=localhost

VM_ID=vm-powerkim01
curl -X GET "http://$RESTSERVER:1024/controlvm/$VM_ID?connection_name=openstack-config01&action=suspend"
