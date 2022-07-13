
echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   NLB: GetVMGroupHealthInfo "
echo "####################################################################"

echo ""

echo "#####---------- GetVMGroupHealthInfo ----------####"
curl -sX GET http://localhost:1024/spider/nlb/spider-nlb-01/health -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'"
	}' |json_pp

