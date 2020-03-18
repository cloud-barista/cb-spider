RESTSERVER=localhost

VM_NAME=CBVm
curl -X GET "http://$RESTSERVER:1024/spider/controlvm/$VM_NAME?connection_name=openstack-config01&action=resume"
