source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
	ID=`curl -X GET http://$RESTSERVER:1024/vnetwork?connection_name=${NAME} |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
	curl -X DELETE http://$RESTSERVER:1024/vnetwork/${ID}?connection_name=${NAME} |json_pp
done
