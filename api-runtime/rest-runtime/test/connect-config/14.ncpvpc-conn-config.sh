RESTSERVER=localhost

 # Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"ncpvpc-driver01","ProviderName":"NCPVPC", "DriverLibFileName":"ncpvpc-driver-v1.0.so"}'

 # Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"ncpvpc-credential01","ProviderName":"NCPVPC", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXXXXXXXXXXXXXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"}]}'

 # Cloud Region & Zone Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncpvpc-korea1","ProviderName":"NCPVPC", "KeyValueInfoList": [{"Key":"Region", "Value":"KR"}, {"Key":"Zone", "Value":"KR-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncpvpc-korea2","ProviderName":"NCPVPC", "KeyValueInfoList": [{"Key":"Region", "Value":"KR"}, {"Key":"Zone", "Value":"KR-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncpvpc-singapore4","ProviderName":"NCPVPC", "KeyValueInfoList": [{"Key":"Region", "Value":"SGN"}, {"Key":"Zone", "Value":"SGN-4"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncpvpc-singapore5","ProviderName":"NCPVPC", "KeyValueInfoList": [{"Key":"Region", "Value":"SGN"}, {"Key":"Zone", "Value":"SGN-5"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncpvpc-japan4","ProviderName":"NCPVPC", "KeyValueInfoList": [{"Key":"Region", "Value":"JPN"}, {"Key":"Zone", "Value":"JPN-4"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncpvpc-japan5","ProviderName":"NCPVPC", "KeyValueInfoList": [{"Key":"Region", "Value":"JPN"}, {"Key":"Zone", "Value":"JPN-5"}]}'

 # Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncpvpc-korea1-config","ProviderName":"NCPVPC", "DriverName":"ncpvpc-driver01", "CredentialName":"ncpvpc-credential01", "RegionName":"ncpvpc-korea1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncpvpc-korea2-config","ProviderName":"NCPVPC", "DriverName":"ncpvpc-driver01", "CredentialName":"ncpvpc-credential01", "RegionName":"ncpvpc-korea2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncpvpc-singapore4-config","ProviderName":"NCPVPC", "DriverName":"ncpvpc-driver01", "CredentialName":"ncpvpc-credential01", "RegionName":"ncpvpc-singapore4"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncpvpc-singapore5-config","ProviderName":"NCPVPC", "DriverName":"ncpvpc-driver01", "CredentialName":"ncpvpc-credential01", "RegionName":"ncpvpc-singapore5"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncpvpc-japan4-config","ProviderName":"NCPVPC", "DriverName":"ncpvpc-driver01", "CredentialName":"ncpvpc-credential01", "RegionName":"ncpvpc-japan4"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncpvpc-japan5-config","ProviderName":"NCPVPC", "DriverName":"ncpvpc-driver01", "CredentialName":"ncpvpc-credential01", "RegionName":"ncpvpc-japan5"}'
