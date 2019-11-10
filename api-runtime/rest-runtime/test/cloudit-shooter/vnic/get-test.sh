source ../setup.env

ID=025e5edc-54ad-4b98-9292-6eeca4c36a6d
curl -X GET http://$RESTSERVER:1024/vnic/$ID?connection_name=cloudit-config01 |json_pp
