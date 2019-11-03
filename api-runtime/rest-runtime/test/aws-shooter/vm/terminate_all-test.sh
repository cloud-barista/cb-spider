source ../setup.env

#echo ========================== ohio
#VM_IDS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-ohio-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
#for VM_ID in ${VM_IDS}
#do
#        echo terminate ${VM_ID} ...
#        curl -X DELETE http://$RESTSERVER:1024/vm/${VM_ID}?connection_name=aws-ohio-config |json_pp
#done

echo ========================== oregon
VM_IDS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-oregon-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for VM_ID in ${VM_IDS}
do
        echo terminate ${VM_ID} ...
        curl -X DELETE http://$RESTSERVER:1024/vm/${VM_ID}?connection_name=aws-oregon-config |json_pp
done

echo ========================== singapore
VM_IDS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-singapore-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for VM_ID in ${VM_IDS}
do
        echo terminate ${VM_ID} ...
        curl -X DELETE http://$RESTSERVER:1024/vm/${VM_ID}?connection_name=aws-singapore-config |json_pp
done

echo ========================== paris
VM_IDS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-paris-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for VM_ID in ${VM_IDS}
do
        echo terminate ${VM_ID} ...
        curl -X DELETE http://$RESTSERVER:1024/vm/${VM_ID}?connection_name=aws-paris-config |json_pp
done

echo ========================== saopaulo
VM_IDS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-saopaulo-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for VM_ID in ${VM_IDS}
do
        echo terminate ${VM_ID} ...
        curl -X DELETE http://$RESTSERVER:1024/vm/${VM_ID}?connection_name=aws-saopaulo-config |json_pp
done


echo ========================== tokyo
VM_IDS=`curl -X GET http://$RESTSERVER:1024/vm?connection_name=aws-tokyo-config |json_pp |grep "\"Id\"" |awk '{print $3}' |sed 's/"//g' |sed 's/,//g'`
for VM_ID in ${VM_IDS}
do
        echo terminate ${VM_ID} ...
        curl -X DELETE http://$RESTSERVER:1024/vm/${VM_ID}?connection_name=aws-tokyo-config |json_pp
done

