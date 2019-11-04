source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
	ID=`curl -X GET http://$RESTSERVER:1024/securitygroup?connection_name=${NAME} |json_pp |grep "\"Id\" :" |awk '{print $3}' |awk '{if(NR==2) print $1}' |sed 's/"//g' |sed 's/,//g'`
	curl -X GET http://$RESTSERVER:1024/securitygroup/${ID}?connection_name=${NAME} |json_pp &
done
