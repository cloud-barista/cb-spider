source ../setup.env

ID1=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-ohio-config |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
ID2=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-ohio-config |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
curl -X POST http://$RESTSERVER:1024/vnic?connection_name=aws-ohio-config -H 'Content-Type: application/json' -d '{ "Name": "vnic01-powerkim", "VNetName": "CB-VNet-powerkim", "SecurityGroupIds": [ "'${ID1}'", "'${ID2}'" ], "PublicIPid": "publicipt01-powerkim" }' |json_pp

ID1=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-oregon-config |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
ID2=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-oregon-config |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
curl -X POST http://$RESTSERVER:1024/vnic?connection_name=aws-oregon-config -H 'Content-Type: application/json' -d '{ "Name": "vnic01-powerkim", "VNetName": "CB-VNet-powerkim", "SecurityGroupIds": [ "'${ID1}'", "'${ID2}'" ], "PublicIPid": "publicipt01-powerkim" }' |json_pp

ID1=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-singapore-config |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
ID2=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-singapore-config |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
curl -X POST http://$RESTSERVER:1024/vnic?connection_name=aws-singapore-config -H 'Content-Type: application/json' -d '{ "Name": "vnic01-powerkim", "VNetName": "CB-VNet-powerkim", "SecurityGroupIds": [ "'${ID1}'", "'${ID2}'" ], "PublicIPid": "publicipt01-powerkim" }' |json_pp

ID1=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-paris-config |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
ID2=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-paris-config |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
curl -X POST http://$RESTSERVER:1024/vnic?connection_name=aws-paris-config -H 'Content-Type: application/json' -d '{ "Name": "vnic01-powerkim", "VNetName": "CB-VNet-powerkim", "SecurityGroupIds": [ "'${ID1}'", "'${ID2}'" ], "PublicIPid": "publicipt01-powerkim" }' |json_pp

ID1=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-saopaulo-config |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
ID2=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-saopaulo-config |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
curl -X POST http://$RESTSERVER:1024/vnic?connection_name=aws-saopaulo-config -H 'Content-Type: application/json' -d '{ "Name": "vnic01-powerkim", "VNetName": "CB-VNet-powerkim", "SecurityGroupIds": [ "'${ID1}'", "'${ID2}'" ], "PublicIPid": "publicipt01-powerkim" }' |json_pp


ID1=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-tokyo-config |json_pp |grep "\"Id\" :" |awk '{print $3}' | head -n 1 |sed 's/"//g' |sed 's/,//g'`
ID2=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=aws-tokyo-config |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
curl -X POST http://$RESTSERVER:1024/vnic?connection_name=aws-tokyo-config -H 'Content-Type: application/json' -d '{ "Name": "vnic01-powerkim", "VNetName": "CB-VNet-powerkim", "SecurityGroupIds": [ "'${ID1}'", "'${ID2}'" ], "PublicIPid": "publicipt01-powerkim" }' |json_pp
