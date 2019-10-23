RESTSERVER=192.168.130.8

#image01 -> 생성되있는 Image이름(mcb-test-img)으로 변경
curl -X GET http://$RESTSERVER:1024/vmimage/im1age01/connection_name=cloudit-config01 |json_pp
