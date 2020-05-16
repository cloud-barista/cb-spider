source ../setup.env


num=0
for NAME in "${CONNECT_NAMES[@]}"
do
	echo $NAME

        curl -sX POST http://$RESTSERVER:1024/spider/vmimage -H 'Content-Type: application/json' \
		-d '{ "ConnectionName": "'${NAME}'", "ReqInfo": { "Name": "'${IMG_IDS[num]}'" } }' | json_pp
        num=`expr $num + 1`
done

