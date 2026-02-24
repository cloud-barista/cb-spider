SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD


echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   NLB: DeleteNLB "
echo "####################################################################"

echo ""


echo "#####---------- DeleteNLB ----------####"
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX DELETE http://localhost:1024/spider/nlb/spider-nlb-01 -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'"
	}' | json_pp

