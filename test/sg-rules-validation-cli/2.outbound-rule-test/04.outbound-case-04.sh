#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|ncp|nhn'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "#############################################"
echo -e "# TEST: $0 "
echo -e "#############################################"

source ../common/setup.env $1
source setup.env $1

echo "============== before AddRules: '${SG_NAME}' --- outbound:TCP/1000/1000"
#### @todo Change this command with spctl
curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX POST http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                "RuleInfoList" :
                        [
                                {
                                        "Direction": "outbound",
                                        "IPProtocol": "TCP",
                                        "FromPort": "1000",
                                        "ToPort": "1000"
                                }
                        ]
                }
        }' |json_pp

echo "============== after AddRules: '${SG_NAME}' --- outbound:TCP/1000/1000"

if [ "$SLEEP" ]; then
        sleep $SLEEP
else
        sleep 10
fi

# print the table header of test results
$(test_result_header $1)


#                   CSP I:TCP-01 I:TCP-02 I:UDP-01 I:ICMP-01 | O:TCP-01 O:TCP-02 O:UDP-01 O:ICMP-01
#./io-traffic-test.sh $1    $2      $3        $4       $5           $6       $7       $8      $9
./io-traffic-test.sh $1    pass    fail      skip     fail         pass     pass     skip    fail

# print the end mesg of test results
$(test_result_tailer)


echo -e "\n\n"

