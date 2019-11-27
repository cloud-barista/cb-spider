RESTSERVER=localhost

LOCS=(`cat aws-regions-list.txt |grep RegionName |awk '{print $2}' |sed 's/",//g' |sed 's/"//g'`)

for REGION in "${LOCS[@]}"
do
	echo $REGION

	curl -X POST http://$RESTSERVER:1024/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-'$REGION'","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"Region", "Value":"'$REGION'"}]}'
	curl -X POST http://$RESTSERVER:1024/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"aws-'$REGION'-config","ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-'$REGION'"}'

done
