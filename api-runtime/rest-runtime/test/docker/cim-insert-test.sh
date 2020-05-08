RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"docker-driver01","ProviderName":"DOCKER", "DriverLibFileName":"docker-driver-v1.0.so"}'

 # for Cloud Credential Info
# APIVersion reference: $docker version or https://docs.docker.com/engine/api/ 
# for AWS:VM:Docker
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"docker-credential01","ProviderName":"DOCKER", "KeyValueInfoList": [{"Key":"Host", "Value":"http://xxx.xxx.xxx.xxx:1004"}, {"Key":"APIVersion", "Value":"v1.40"}]}'
# for powerkim:NAS:Docker
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"docker-credential01","ProviderName":"DOCKER", "KeyValueInfoList": [{"Key":"Host", "Value":"http://xxx.xxx.xxx.xxx:1004"}, {"Key":"APIVersion", "Value":"v1.39"}]}'

 # Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"docker-region01","ProviderName":"DOCKER", "KeyValueInfoList": [{"Key":"Region", "Value":"default"}]}'

 # Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"docker-config01","ProviderName":"DOCKER", "DriverName":"docker-driver01", "CredentialName":"docker-credential01", "RegionName":"docker-region01"}'
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"docker-config02","ProviderName":"DOCKER", "DriverName":"docker-driver01", "CredentialName":"docker-credential02", "RegionName":"docker-region01"}'
