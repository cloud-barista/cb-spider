API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD


echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   NLB:   ChangeListener -> ChangeVMGroupInfo -> ChangeHealthCheckerInfo"
echo "##   NLB:   GetNLB	"
echo "####################################################################"

echo ""

echo "#####---------- ChangeListener ----------####"
curl -u $API_USERNAME:$API_PASSWORD -sX PUT http://localhost:1024/spider/nlb/spider-nlb-01/listener -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": {
			"Protocol" : "TCP",
			"Port" : "22"
		}
	}' | json_pp

echo "#####---------- ChangeVMGroup ----------####"
curl -u $API_USERNAME:$API_PASSWORD -sX PUT http://localhost:1024/spider/nlb/spider-nlb-01/vmgroup -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Protocol" : "TCP",
                        "Port" : "22"
                }
        }' | json_pp

echo "#####---------- ChangeHealthChecker ----------####"
curl -u $API_USERNAME:$API_PASSWORD -sX PUT http://localhost:1024/spider/nlb/spider-nlb-01/healthchecker -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Protocol" : "TCP",
                        "Port" : "22",
                        "Interval" : "11",
                        "Timeout" : "11",
                        "Threshold" : "4"
                }
        }' | json_pp


if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

echo "#####---------- GetNLB ----------####"
curl -u $API_USERNAME:$API_PASSWORD -sX GET http://localhost:1024/spider/nlb/spider-nlb-01 -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'"
	}' |json_pp

