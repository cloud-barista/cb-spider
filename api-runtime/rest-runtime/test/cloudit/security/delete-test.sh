RESTSERVER=192.168.130.8

#mcb-test-sg -> sec ID로 변경?
curl -X DELETE http://$RESTSERVER:1024/securitygroup/mcb-test-sg?connection_name=cloudit-config01
