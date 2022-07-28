
echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   NLB: CreateNLB "
echo "####################################################################"

echo ""


PRTL1=$1
PORT1=$2

PRTL2=$3
PORT2=$4

PRTL3=$5
PORT3=$6

INTERVAL=$7
TIMEOUT=$8
THRESHOLD=$9

echo "#####---------- CreateNLB ----------####"
curl -sX POST http://localhost:1024/spider/nlb -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": {
			"Name": "spider-nlb-01", 
			"VPCName": "vpc-01", 
			"Type": "PUBLIC", 
			"Scope": "REGION", 
			"Listener": {
				"Protocol" : "'${PRTL1}'",
	       			"Port" : "'${PORT1}'"
			},
		        "VMGroup": {
	       			"Protocol" : "'${PRTL2}'", 	       
	       			"Port" : "'${PORT2}'", 	       
	       			"VMs" : ["vm-01", "vm-02"]
		 	}, 
			"HealthChecker": {
	       			"Protocol" : "'${PRTL3}'", 	       
	       			"Port" : "'${PORT3}'", 	       

	       			"Interval" : "'${INTERVAL}'", 	       
	       			"Timeout" : "'${TIMEOUT}'", 	       
	       			"Threshold" : "'${THRESHOLD}'" 

			}
		}
	}' | json_pp

