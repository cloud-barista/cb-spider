source ../setup.env

for NAME in "${CONNECT_NAMES[@]}"
do
	echo ========================== $NAME
	curl -X GET http://$RESTSERVER:1024/vmstatus?connection_name=$NAME |json_pp
done
