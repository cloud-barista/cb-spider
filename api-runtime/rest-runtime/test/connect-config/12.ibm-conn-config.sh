RESTSERVER=localhost

 # Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"ibm-driver01","ProviderName":"IBM", "DriverLibFileName":"ibm-driver-v1.0.so"}'

 # Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"ibm-credential01","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"ApiKey", "Value":"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"}]}'


 # Cloud Region & Zone Info

# Multizone regions
# Americas - Dallas
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-us-south-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-south"}, {"Key":"Zone", "Value":"us-south-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-us-south-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-south"}, {"Key":"Zone", "Value":"us-south-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-us-south-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-south"}, {"Key":"Zone", "Value":"us-south-3"}]}'

# Americas - Sao Paulo
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-br-sao-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"br-sao"}, {"Key":"Zone", "Value":"br-sao-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-br-sao-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"br-sao"}, {"Key":"Zone", "Value":"br-sao-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-br-sao-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"br-sao"}, {"Key":"Zone", "Value":"br-sao-3"}]}'

# Americas - Toronto
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-ca-tor-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"ca-tor"}, {"Key":"Zone", "Value":"ca-tor-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-ca-tor-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"ca-tor"}, {"Key":"Zone", "Value":"ca-tor-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-ca-tor-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"ca-tor"}, {"Key":"Zone", "Value":"ca-tor-3"}]}'

# Americas - Washington DC
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-us-east-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-east"}, {"Key":"Zone", "Value":"us-east-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-us-east-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-east"}, {"Key":"Zone", "Value":"us-east-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-us-east-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"us-east"}, {"Key":"Zone", "Value":"us-east-3"}]}'

# Europe - Frankfurt
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-eu-de-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-de"}, {"Key":"Zone", "Value":"eu-de-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-eu-de-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-de"}, {"Key":"Zone", "Value":"eu-de-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-eu-de-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-de"}, {"Key":"Zone", "Value":"eu-de-3"}]}'

# Europe - London
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-eu-gb-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-gb"}, {"Key":"Zone", "Value":"eu-gb-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-eu-gb-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-gb"}, {"Key":"Zone", "Value":"eu-gb-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-eu-gb-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-gb"}, {"Key":"Zone", "Value":"eu-gb-3"}]}'

# Asia Pacific - Osaka
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-jp-osa-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-osa"}, {"Key":"Zone", "Value":"jp-osa-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-jp-osa-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-osa"}, {"Key":"Zone", "Value":"jp-osa-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-jp-osa-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-osa"}, {"Key":"Zone", "Value":"jp-osa-3"}]}'

# Asia Pacific - Sydney
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-au-syd-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"au-syd"}, {"Key":"Zone", "Value":"au-syd-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-au-syd-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"au-syd"}, {"Key":"Zone", "Value":"au-syd-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-au-syd-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"au-syd"}, {"Key":"Zone", "Value":"au-syd-3"}]}'

# Asia Pacific - Tokyo
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-jp-tok-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-tok"}, {"Key":"Zone", "Value":"jp-tok-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-jp-tok-2","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-tok"}, {"Key":"Zone", "Value":"jp-tok-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-jp-tok-3","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"jp-tok"}, {"Key":"Zone", "Value":"jp-tok-3"}]}'

# Single-zone regions
# Asia Pacific - Seoul
#curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-kr-seo-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"kr-seo"}, {"Key":"Zone", "Value":"kr-seo-1"}]}'

# Asia Pacific - Chennai
#curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"ibm-in-che-1","ProviderName":"IBM", "KeyValueInfoList": [{"Key":"Region", "Value":"in-che"}, {"Key":"Zone", "Value":"in-che-1"}]}'


 # Cloud Connection Config Info
 # Americas - Dallas 
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-us-south-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-us-south-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-us-south-2-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-us-south-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-us-south-3-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-us-south-3"}'

# Americas - Sao Paulo
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-br-sao-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-br-sao-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-br-sao-2-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-br-sao-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-br-sao-3-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-br-sao-3"}'

# Americas - Toronto
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-ca-tor-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-ca-tor-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-ca-tor-2-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-ca-tor-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-ca-tor-3-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-ca-tor-3"}'

# Americas - Washington DC
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-us-east-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-us-east-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-us-east-2-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-us-east-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-us-east-3-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-us-east-3"}'

# Europe - Frankfurt
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-eu-de-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-eu-de-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-eu-de-2-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-eu-de-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-eu-de-3-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-eu-de-3"}'

# Europe - London
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-eu-gb-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-eu-gb-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-eu-gb-2-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-eu-gb-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-eu-gb-3-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-eu-gb-3"}'

# Asia Pacific - Osaka
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-jp-osa-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-jp-osa-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-jp-osa-2-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-jp-osa-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-jp-osa-3-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-jp-osa-3"}'

# Asia Pacific - Sydney
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-au-syd-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-au-syd-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-au-syd-2-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-au-syd-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-au-syd-3-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-au-syd-3"}'

# Asia Pacific - Tokyo
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-jp-tok-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-jp-tok-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-jp-tok-2-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-jp-tok-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-jp-tok-3-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-jp-tok-3"}'

# Asia Pacific - Seoul
#curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-kr-seo-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-kr-seo-1"}'

# Asia Pacific - Chennai
#curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"ibm-in-che-1-config","ProviderName":"IBM", "DriverName":"ibm-driver01", "CredentialName":"ibm-credential01", "RegionName":"ibm-in-che-1"}'
