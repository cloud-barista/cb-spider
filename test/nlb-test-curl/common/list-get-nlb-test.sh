API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD


echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   NLB: ListNLB -> GetNLB "
echo "####################################################################"

echo ""


echo "#####---------- ListNLB ----------####"
curl -u $API_USERNAME:$API_PASSWORD -sX GET http://localhost:1024/spider/nlb -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'"
	}' |json_pp


echo "#####---------- GetNLB ----------####"
curl -u $API_USERNAME:$API_PASSWORD -sX GET http://localhost:1024/spider/nlb/spider-nlb-01 -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'"
	}' |json_pp

echo "#####---------- ListAllNLB ----------####"
curl -u $API_USERNAME:$API_PASSWORD -sX GET http://localhost:1024/spider/allnlb -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp
