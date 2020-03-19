RESTSERVER=localhost

curl -X GET http://$RESTSERVER:1024/spider/vmstatus?connection_name=openstack-config01  |json_pp
