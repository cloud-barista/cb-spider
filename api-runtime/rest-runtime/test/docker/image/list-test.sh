source ../setup.env

num=0
for CONN_CONFIG in "${CONNECT_NAMES[@]}"
do
        curl -sX GET http://$RESTSERVER:1024/spider/vmimage -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp 
        num=`expr $num + 1`
done

