

if [ "$1" = "" ]; then
        max=2
else
        max=$1
fi

echo "####################################################################"
echo "## Full Test Scripts for CB-Spider IID Working Version - 2020.04.21."
echo "##   4. VM: Terminate($max)"
echo "##   3. KeyPair: Delete"
echo "##   2. SecurityGroup: Delete"
echo "##   1. VPC: Delete"
echo "####################################################################"

echo "####################################################################"
echo "####################################################################"
echo "####################################################################"

echo "####################################################################"
echo "## 4. VM: Terminate($max)"
echo "####################################################################"
for (( num=1; num <= $max; num++ ))
do
        curl -sX DELETE http://localhost:1024/spider/vm/powerkim-vm-test-${num} -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp &
done


echo "============== sleep 300 after delete VM"
sleep 300


echo "####################################################################"
echo "## 3. KeyPair: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/keypair/keypair-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "####################################################################"
echo "## 2. SecurityGroup: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/securitygroup/sg-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp
echo "####################################################################"
echo "## 1. VPC: Delete"
echo "####################################################################"
curl -sX DELETE http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}' |json_pp

