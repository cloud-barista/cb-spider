RESTSERVER=192.168.130.8

curl -X POST http://$RESTSERVER:1024/vnic?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{
"VmId": "025e5edc-54ad-4b98-9292-6eeca4c36a6d",
"SubnetAddr": "10.0.8.0",
"Secgroups":
[
"ID", "b2be62e7-fd29-43ff-b008-08ae736e092a"
],
"Type": "INTERNAL",
"IP" : ""
}'
