source ../setup.env


echo ========================== tokyo
PUBLIC_IPS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-tokyo-config |json_pp |grep "\"PublicIP\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for PUBLIC_IP in ${PUBLIC_IPS}
do
        echo tokyo: copy testsvc into ${PUBLIC_IP} ...
        scp -i ../keypair/aws-tokyo.key -o "StrictHostKeyChecking no" ./testsvc/TESTSvc ./testsvc/setup.env ubuntu@$PUBLIC_IP:/tmp
        scp -i ../keypair/aws-tokyo.key -o "StrictHostKeyChecking no" -r ./testsvc/conf ubuntu@$PUBLIC_IP:/tmp
        ssh -i ../keypair/aws-tokyo.key -o "StrictHostKeyChecking no" ubuntu@$PUBLIC_IP 'source /tmp/setup.env;/tmp/TESTSvc'
done

