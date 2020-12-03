source setup.env

curl -X POST "http://$RESTSERVER:1024/spider/driver" -H 'Content-Type: application/json' -d '{"DriverName":"cloudtwin-driver01","ProviderName":"CLOUDTWIN", "DriverLibFileName":"cloudtwin-driver-v1.0.so"}'

 # for Cloud Region Info
curl -X POST "http://$RESTSERVER:1024/spider/region" -H 'Content-Type: application/json' -d '{"RegionName":"cloudtwin-region","ProviderName":"CLOUDTWIN", "KeyValueInfoList": [{"Key":"Region", "Value":"default"}]}'


num=0
for NAME in "${CLOUD_NAMES[@]}"
do
	#CONNECT_NAME=cloudtwin-$NAME-config

	num=`expr $num + 1`

	echo ================= cloud-$num
	 # for Cloud Credential Info
	curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{
	    "CredentialName":"cloudtwin-cloud-'$num'-credential01",
	    "ProviderName":"CLOUDTWIN",
	    "KeyValueInfoList": [
		{"Key":"IdentityEndpoint", "Value":"http://XX.XX.XX.XX:8080"},
		{"Key":"DomainName", "Value":"cloud-'$num'"},
		{"Key":"MockName", "Value":"mock_name01"}
	]}'

	 # for Cloud Connection Config Info
	curl -X POST "http://$RESTSERVER:1024/spider/connectionconfig" -H 'Content-Type: application/json' -d '{"ConfigName":"cloudtwin-cloud-'$num'-config","ProviderName":"CLOUDTWIN", "DriverName":"cloudtwin-driver01", "CredentialName":"cloudtwin-cloud-'$num'-credential01", "RegionName":"cloudtwin-region"}'

done


