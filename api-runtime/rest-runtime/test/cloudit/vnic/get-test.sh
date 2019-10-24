RESTSERVER=192.168.130.8

# vNetName -> vNicId로 변경
curl -X GET http://$RESTSERVER:1024/vnic/025e5edc-54ad-4b98-9292-6eeca4c36a6d?connection_name=cloudit-config01
