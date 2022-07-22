
echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   NLB:   AddVMs -> GetVMGroupHealthInfo -> RemoveVMs -> GetVMGroupHealthInfo"
echo "####################################################################"

echo
echo "##########################################"
echo "#### prepare to add VMs into VMGroup  ####"
echo "##########################################"


echo "#####---------- RemoveVMs ----------####"
echo
curl -sX DELETE http://localhost:1024/spider/nlb/spider-nlb-01/vms -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "VMs" : ["vm-03", "vm-04"]
                }
        }' | json_pp
echo

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

echo "#####---------- GetNLB ----------####"
curl -sX GET http://localhost:1024/spider/nlb/spider-nlb-01 -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp

echo
echo "##########################################"
echo "#### Teminate VMs used VMGroup testing ####"
echo "##########################################"
echo

echo "#####---------- TerminateVM:vm-03 ----------####"
curl -sX DELETE http://localhost:1024/spider/vm/vm-03 -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'"
	}' |json_pp

if [ "$SLEEP" ]; then
        sleep $SLEEP
fi

echo "#####---------- TerminateVM:vm-04 ----------####"
curl -sX DELETE http://localhost:1024/spider/vm/vm-04 -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp

