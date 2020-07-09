source ../setup.env


num=0
for NAME in "${CONNECT_NAMES[@]}"
do
	echo $NAME

        # escape: "/" => "%2F"
        IMG_IDS[num]=`echo ${IMG_IDS[num]} | sed 's/\//%2F/g'`

        curl -sX GET http://$RESTSERVER:1024/spider/vmimage/${IMG_IDS[num]} -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${NAME}'" }' | json_pp
        num=`expr $num + 1`
done

