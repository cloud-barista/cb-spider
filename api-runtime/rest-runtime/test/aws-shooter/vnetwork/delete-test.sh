RESTSERVER=localhost

ID=`curl -X GET http://$RESTSERVER:1024/vnetwork?connection_name=aws-ohio-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
curl -X DELETE http://$RESTSERVER:1024/vnetwork/${ID}?connection_name=aws-ohio-config |json_pp

ID=`curl -X GET http://$RESTSERVER:1024/vnetwork?connection_name=aws-oregon-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
curl -X DELETE http://$RESTSERVER:1024/vnetwork/${ID}?connection_name=aws-oregon-config |json_pp

ID=`curl -X GET http://$RESTSERVER:1024/vnetwork?connection_name=aws-singapore-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
curl -X DELETE http://$RESTSERVER:1024/vnetwork/${ID}?connection_name=aws-singapore-config |json_pp

ID=`curl -X GET http://$RESTSERVER:1024/vnetwork?connection_name=aws-paris-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
curl -X DELETE http://$RESTSERVER:1024/vnetwork/${ID}?connection_name=aws-paris-config |json_pp

ID=`curl -X GET http://$RESTSERVER:1024/vnetwork?connection_name=aws-saopaulo-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
curl -X DELETE http://$RESTSERVER:1024/vnetwork/${ID}?connection_name=aws-saopaulo-config |json_pp


ID=`curl -X GET http://$RESTSERVER:1024/vnetwork?connection_name=aws-tokyo-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
curl -X DELETE http://$RESTSERVER:1024/vnetwork/${ID}?connection_name=aws-tokyo-config |json_pp
