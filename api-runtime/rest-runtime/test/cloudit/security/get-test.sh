RESTSERVER=192.168.130.8

#security01 -> sec ID로 변경
curl -X GET http://$RESTSERVER:1024/securitygroup/mcb-test-sg?connection_name=cloudit-config01
