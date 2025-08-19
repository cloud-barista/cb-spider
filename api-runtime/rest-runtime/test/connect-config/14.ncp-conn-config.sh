RESTSERVER=localhost

 # Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"ncp-driver01","ProviderName":"NCP", "DriverLibFileName":"ncp-driver-v1.0.so"}'

 # Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"ncp-credential01","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXXXXXXXXXXXXXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"}]}'

 # Cloud Region & Zone Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-korea1","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"KR"}, {"Key":"Zone", "Value":"KR-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-korea2","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"KR"}, {"Key":"Zone", "Value":"KR-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-singapore4","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"SGN"}, {"Key":"Zone", "Value":"SGN-4"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-singapore5","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"SGN"}, {"Key":"Zone", "Value":"SGN-5"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-japan4","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"JPN"}, {"Key":"Zone", "Value":"JPN-4"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ncp-japan5","ProviderName":"NCP", "KeyValueInfoList": [{"Key":"Region", "Value":"JPN"}, {"Key":"Zone", "Value":"JPN-5"}]}'

 # Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-korea1-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-korea1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-korea2-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-korea2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-singapore4-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-singapore4"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-singapore5-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-singapore5"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-japan4-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-japan4"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ncp-japan5-config","ProviderName":"NCP", "DriverName":"ncp-driver01", "CredentialName":"ncp-credential01", "RegionName":"ncp-japan5"}'
