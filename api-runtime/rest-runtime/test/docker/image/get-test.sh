source ../setup.env


num=0
for NAME in "${CONNECT_NAMES[@]}"
do
	echo $NAME

        curl -sX GET http://$RESTSERVER:1024/spider/vmimage/${IMG_IDS[num]} -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${NAME}'" }' | json_pp
        num=`expr $num + 1`
done

