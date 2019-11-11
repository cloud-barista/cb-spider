RESTSERVER=localhost

VM_ID=3cac5ef3-a338-411d-9614-c42b044fe19c
curl -X GET "http://$RESTSERVER:1024/controlvm/$VM_ID?connection_name=openstack-config01&action=resume"
