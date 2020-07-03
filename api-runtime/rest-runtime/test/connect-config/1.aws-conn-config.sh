RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"aws-driver01","ProviderName":"AWS", "DriverLibFileName":"aws-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"aws-credential01","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXX"}]}'

 # Cloud Region Info for Shooter
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-ohio","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"Region", "Value":"us-east-2"}, {"Key":"Zone", "Value":"us-east-2a"}]}'
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-oregon","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"Region", "Value":"us-west-2"}, {"Key":"Zone", "Value":"us-west-2a"}]}'
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-singapore","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-southeast-1"}, {"Key":"Zone", "Value":"ap-southeast-1a"}]}'
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-paris","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-west-3"}, {"Key":"Zone", "Value":"eu-west-3a"}]}'
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-saopaulo","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"Region", "Value":"sa-east-1"}, {"Key":"Zone", "Value":"sa-east-1a"}]}'

 # for test service
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-tokyo","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-northeast-1"}, {"Key":"Zone", "Value":"ap-northeast-1a"}]}'

 # Cloud Connection Config Info for Shooter
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"aws-ohio-config","ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-ohio"}'
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"aws-oregon-config","ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-oregon"}'
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"aws-singapore-config","ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-singapore"}'
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"aws-paris-config","ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-paris"}'
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"aws-saopaulo-config","ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-saopaulo"}'

 # for test service
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"aws-tokyo-config","ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-tokyo"}'

