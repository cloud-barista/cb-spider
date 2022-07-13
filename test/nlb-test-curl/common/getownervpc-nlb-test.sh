
echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   NLB: GetNLBOwnerVPC "
echo "####################################################################"

echo ""

if [ "$1" = "" ]; then
	echo
	echo -e 'usage: '$0' cspid'
	echo
	exit 0;	
fi

echo "#####---------- GetNLBOwnerVPC ----------####"
curl -sX GET http://localhost:1024/spider/getnlbowner -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "CSPId": "'$1'"
		}
	}' |json_pp

