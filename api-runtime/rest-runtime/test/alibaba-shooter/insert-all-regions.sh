RESTSERVER=localhost

LOCS=(`cat alibaba-regions-list.txt |grep RegionId |awk '{print $2}' |sed 's/",//g' |sed 's/"//g'`)

for REGION in "${LOCS[@]}"
do
	echo $REGION

	curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"alibaba-'$REGION'","ProviderName":"ALIBABA", "KeyValueInfoList": [{"Key":"Region", "Value":"'$REGION'"}]}'
	curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"alibaba-'$REGION'-config","ProviderName":"ALIBABA", "DriverName":"alibaba-driver01", "CredentialName":"alibaba-credential01", "RegionName":"alibaba-'$REGION'"}'

done
