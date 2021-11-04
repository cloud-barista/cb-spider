RESTSERVER=localhost

 # Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"ibmvpc-driver01","ProviderName":"IBM", "DriverLibFileName":"ibmvpc-driver-v1.0.so"}'

 # Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"ibmvpc-credential01","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"ApiKey", "Value":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"}]}'


 # Cloud Region & Zone Info

# Multizone regions
# Americas - Dallas
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-us-south-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-south"}, {"Key":"Zone", "Value":"us-south-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-us-south-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-south"}, {"Key":"Zone", "Value":"us-south-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-us-south-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-south"}, {"Key":"Zone", "Value":"us-south-3"}]}'

# Americas - Sao Paulo
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-br-sao-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"br-sao"}, {"Key":"Zone", "Value":"br-sao-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-br-sao-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"br-sao"}, {"Key":"Zone", "Value":"br-sao-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-br-sao-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"br-sao"}, {"Key":"Zone", "Value":"br-sao-3"}]}'

# Americas - Toronto
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-ca-tor-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"ca-tor"}, {"Key":"Zone", "Value":"ca-tor-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-ca-tor-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"ca-tor"}, {"Key":"Zone", "Value":"ca-tor-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-ca-tor-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"ca-tor"}, {"Key":"Zone", "Value":"ca-tor-3"}]}'

# Americas - Washington DC
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-us-east-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-east"}, {"Key":"Zone", "Value":"us-east-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-us-east-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-east"}, {"Key":"Zone", "Value":"us-east-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-us-east-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-east"}, {"Key":"Zone", "Value":"us-east-3"}]}'

# Europe - Frankfurt
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-eu-de-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-de"}, {"Key":"Zone", "Value":"eu-de-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-eu-de-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-de"}, {"Key":"Zone", "Value":"eu-de-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-eu-de-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-de"}, {"Key":"Zone", "Value":"eu-de-3"}]}'

# Europe - London
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-eu-gb-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-gb"}, {"Key":"Zone", "Value":"eu-gb-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-eu-gb-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-gb"}, {"Key":"Zone", "Value":"eu-gb-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-eu-gb-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-gb"}, {"Key":"Zone", "Value":"eu-gb-3"}]}'

# Asia Pacific - Osaka
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-jp-osa-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-osa"}, {"Key":"Zone", "Value":"jp-osa-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-jp-osa-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-osa"}, {"Key":"Zone", "Value":"jp-osa-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-jp-osa-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-osa"}, {"Key":"Zone", "Value":"jp-osa-3"}]}'

# Asia Pacific - Sydney
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-au-syd-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"au-syd"}, {"Key":"Zone", "Value":"au-syd-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-au-syd-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"au-syd"}, {"Key":"Zone", "Value":"au-syd-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-au-syd-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"au-syd"}, {"Key":"Zone", "Value":"au-syd-3"}]}'

# Asia Pacific - Tokyo
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-jp-tok-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-tok"}, {"Key":"Zone", "Value":"jp-tok-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-jp-tok-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-tok"}, {"Key":"Zone", "Value":"jp-tok-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-jp-tok-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-tok"}, {"Key":"Zone", "Value":"jp-tok-3"}]}'

# Single-zone regions
# Asia Pacific - Seoul
#curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-kr-seo-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"kr-seo"}, {"Key":"Zone", "Value":"kr-seo-1"}]}'

# Asia Pacific - Chennai
#curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibmvpc-in-che-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"in-che"}, {"Key":"Zone", "Value":"in-che-1"}]}'


 # Cloud Connection Config Info
 # Americas - Dallas 
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-us-south-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-us-south-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-us-south-2-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-us-south-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-us-south-3-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-us-south-3"}'

# Americas - Sao Paulo
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-br-sao-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-br-sao-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-br-sao-2-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-br-sao-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-br-sao-3-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-br-sao-3"}'

# Americas - Toronto
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-ca-tor-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-ca-tor-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-ca-tor-2-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-ca-tor-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-ca-tor-3-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-ca-tor-3"}'

# Americas - Washington DC
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-us-east-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-us-east-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-us-east-2-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-us-east-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-us-east-3-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-us-east-3"}'

# Europe - Frankfurt
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-eu-de-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-eu-de-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-eu-de-2-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-eu-de-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-eu-de-3-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-eu-de-3"}'

# Europe - London
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-eu-gb-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-eu-gb-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-eu-gb-2-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-eu-gb-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-eu-gb-3-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-eu-gb-3"}'

# Asia Pacific - Osaka
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-jp-osa-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-jp-osa-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-jp-osa-2-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-jp-osa-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-jp-osa-3-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-jp-osa-3"}'

# Asia Pacific - Sydney
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-au-syd-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-au-syd-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-au-syd-2-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-au-syd-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-au-syd-3-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-au-syd-3"}'

# Asia Pacific - Tokyo
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-jp-tok-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-jp-tok-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-jp-tok-2-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-jp-tok-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-jp-tok-3-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-jp-tok-3"}'

# Asia Pacific - Seoul
#curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-kr-seo-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-kr-seo-1"}'

# Asia Pacific - Chennai
#curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibmvpc-in-che-1-config","ProviderName":"IBM", "DriverName":"ibmvpc-driver01", "CredentialName":"ibmvpc-credential01", "RegionName":"ibmvpc-in-che-1"}'
