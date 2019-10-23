RESTSERVER=localhost

VM_ID=f565d481-6209-4932-a422-04195d4215e0
curl -X GET "http://$RESTSERVER:1024/controlvm/$VM_ID?connection_name=openstack-config01&action=suspend"
