
echo "####################################################################"
echo "## NLB Test Scripts for CB-Spider - 2022.06."
echo "##   NLB:   AddVMs -> GetVMGroupHealthInfo -> RemoveVMs -> GetVMGroupHealthInfo"
echo "####################################################################"

echo ""


#
#-------------------- prepare VMs
#
echo "#####---------- StartVM:vm-03 ----------####"
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Name": "vm-03",
                        "ImageName": "'${IMAGE_NAME}'",
                        "VMSpecName": "'${SPEC_NAME}'",
                        "VPCName": "vpc-01",
                        "SubnetName": "subnet-01",
                        "SecurityGroupNames": [ "sg-01" ],
                        "KeyPairName": "keypair-01"
                }
        }' |json_pp

echo "#####---------- StartVM:vm-04 ----------####"
curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                        "Name": "vm-04",
                        "ImageName": "'${IMAGE_NAME}'",
                        "VMSpecName": "'${SPEC_NAME}'",
                        "VPCName": "vpc-01",
                        "SubnetName": "subnet-01",
                        "SecurityGroupNames": [ "sg-01" ],
                        "KeyPairName": "keypair-01"
                }
        }' |json_pp


#################################################
#################################################
#################################################

echo "#####---------- AddVMs ----------####"
curl -sX POST http://localhost:1024/spider/nlb/spider-nlb-01/vms -H 'Content-Type: application/json' -d \
	'{
		"ConnectionName": "'${CONN_CONFIG}'", 
		"ReqInfo": {
			"VMs" : ["vm-03", "vm-04"]
		}
	}' | json_pp


echo "#####---------- GetNLB ----------####"
curl -sX GET http://localhost:1024/spider/nlb/spider-nlb-01 -H 'Content-Type: application/json' -d \
	'{ 
		"ConnectionName": "'${CONN_CONFIG}'"
	}' |json_pp


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

echo "#####---------- GetNLB ----------####"
curl -sX GET http://localhost:1024/spider/nlb/spider-nlb-01 -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'"
        }' |json_pp

