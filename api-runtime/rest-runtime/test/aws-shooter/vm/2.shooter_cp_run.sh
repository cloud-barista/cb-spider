source ../setup.env

#scp -i ../keypair/aws-singapore.key -o "StrictHostKeyChecking no" ./shooter/shooter-sample.sh node13:/tmp
#ssh -i ../keypair/aws-singapore.key -o "StrictHostKeyChecking no" ubuntu@13.228.136.25 

# echo ========================== ohio
# PUBLIC_IPS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-ohio-config |json_pp |grep "\"PublicIP\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
# for PUBLIC_IP in ${PUBLIC_IPS}
# do
#         echo ohio: copy shooter into ${PUBLIC_IP} ...
# 	scp -i ../keypair/aws-ohio.key -o "StrictHostKeyChecking no" ./shooter/shooter.sh ubuntu@$PUBLIC_IP:/tmp
# 	ssh -i ../keypair/aws-ohio.key -o "StrictHostKeyChecking no" ubuntu@$PUBLIC_IP /tmp/shooter.sh
# done

echo ========================== oregon
PUBLIC_IPS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-oregon-config |json_pp |grep "\"PublicIP\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for PUBLIC_IP in ${PUBLIC_IPS}
do
        echo oregon: copy shooter into ${PUBLIC_IP} ...
        scp -i ../keypair/aws-oregon.key -o "StrictHostKeyChecking no" ./shooter/shooter.sh ubuntu@$PUBLIC_IP:/tmp
        ssh -i ../keypair/aws-oregon.key -o "StrictHostKeyChecking no" ubuntu@$PUBLIC_IP /tmp/shooter.sh
done
echo ========================== singapore
PUBLIC_IPS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-singapore-config |json_pp |grep "\"PublicIP\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for PUBLIC_IP in ${PUBLIC_IPS}
do
        echo singapore: copy shooter into ${PUBLIC_IP} ...
        scp -i ../keypair/aws-singapore.key -o "StrictHostKeyChecking no" ./shooter/shooter.sh ubuntu@$PUBLIC_IP:/tmp
        ssh -i ../keypair/aws-singapore.key -o "StrictHostKeyChecking no" ubuntu@$PUBLIC_IP /tmp/shooter.sh
done

echo ========================== paris
PUBLIC_IPS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-paris-config |json_pp |grep "\"PublicIP\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for PUBLIC_IP in ${PUBLIC_IPS}
do
        echo paris: copy shooter into ${PUBLIC_IP} ...
        scp -i ../keypair/aws-paris.key -o "StrictHostKeyChecking no" ./shooter/shooter.sh ubuntu@$PUBLIC_IP:/tmp
        ssh -i ../keypair/aws-paris.key -o "StrictHostKeyChecking no" ubuntu@$PUBLIC_IP /tmp/shooter.sh
done

echo ========================== saopaulo
PUBLIC_IPS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-saopaulo-config |json_pp |grep "\"PublicIP\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for PUBLIC_IP in ${PUBLIC_IPS}
do
        echo saopaulo: copy shooter into ${PUBLIC_IP} ...
        scp -i ../keypair/aws-saopaulo.key -o "StrictHostKeyChecking no" ./shooter/shooter.sh ubuntu@$PUBLIC_IP:/tmp
        ssh -i ../keypair/aws-saopaulo.key -o "StrictHostKeyChecking no" ubuntu@$PUBLIC_IP /tmp/shooter.sh
done

