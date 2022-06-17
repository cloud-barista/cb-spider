source mock.env

nlbinfo=`curl -sX GET http://localhost:1024/spider/nlb/spider-nlb-01 -H 'Content-Type: application/json' -d \
	'{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp`

cspid=`echo -e "$nlbinfo" |grep SystemId |grep nl |sed -e 's/"SystemId" : "//g' | sed -e 's/",//g' | sed -e 's/"//g'`


../common/getownervpc-nlb-test.sh $cspid
