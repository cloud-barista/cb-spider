RESTSERVER=localhost

# Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json' -d '{"DriverName":"tencent-driver01","ProviderName":"TENCENT", "DriverLibFileName":"tencent-driver-v1.0.so"}'

# Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"tencent-credential01","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXXX"}, {"Key":"ClientSecret", "Value":"XXXX"}]}'


### Cloud Region & Zone Info
# South China (Guangzhou) : ap-guangzhou  $$$(Caution!!) ap-guangzhou-5 zone은 없음.
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-guangzhou-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-guangzhou"}, {"Key":"Zone", "Value":"ap-guangzhou-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-guangzhou-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-guangzhou"}, {"Key":"Zone", "Value":"ap-guangzhou-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-guangzhou-3","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-guangzhou"}, {"Key":"Zone", "Value":"ap-guangzhou-3"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-guangzhou-4","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-guangzhou"}, {"Key":"Zone", "Value":"ap-guangzhou-4"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-guangzhou-5","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-guangzhou"}, {"Key":"Zone", "Value":"ap-guangzhou-5"}]}'

# East China (Shanghai) : ap-shanghai
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-shanghai-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-shanghai"}, {"Key":"Zone", "Value":"ap-shanghai-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-shanghai-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-shanghai"}, {"Key":"Zone", "Value":"ap-shanghai-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-shanghai-3","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-shanghai"}, {"Key":"Zone", "Value":"ap-shanghai-3"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-shanghai-4","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-shanghai"}, {"Key":"Zone", "Value":"ap-shanghai-4"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-shanghai-5","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-shanghai"}, {"Key":"Zone", "Value":"ap-shanghai-5"}]}'

# East China (Nanjing) : ap-nanjing
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-nanjing-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-nanjing"}, {"Key":"Zone", "Value":"ap-nanjing-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-nanjing-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-nanjing"}, {"Key":"Zone", "Value":"ap-nanjing-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-nanjing-3","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-nanjing"}, {"Key":"Zone", "Value":"ap-nanjing-3"}]}'

# North China (Beijing) : ap-beijing
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-beijing-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-beijing"}, {"Key":"Zone", "Value":"ap-beijing-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-beijing-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-beijing"}, {"Key":"Zone", "Value":"ap-beijing-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-beijing-3","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-beijing"}, {"Key":"Zone", "Value":"ap-beijing-3"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-beijing-4","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-beijing"}, {"Key":"Zone", "Value":"ap-beijing-4"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-beijing-5","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-beijing"}, {"Key":"Zone", "Value":"ap-beijing-5"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-beijing-6","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-beijing"}, {"Key":"Zone", "Value":"ap-beijing-6"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-beijing-7","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-beijing"}, {"Key":"Zone", "Value":"ap-beijing-7"}]}'

# Southwest China (Chengdu) : ap-chengdu
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-chengdu-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-chengdu"}, {"Key":"Zone", "Value":"ap-chengdu-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-chengdu-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-chengdu"}, {"Key":"Zone", "Value":"ap-chengdu-2"}]}'

# Southwest China (Chongqing) : ap-chongqing
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-chongqing-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-chongqing"}, {"Key":"Zone", "Value":"ap-chongqing-1"}]}'

# Hong Kong/Macao/Taiwan (Hong Kong, China) : ap-hongkong
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-hongkong-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-hongkong"}, {"Key":"Zone", "Value":"ap-hongkong-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-hongkong-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-hongkong"}, {"Key":"Zone", "Value":"ap-hongkong-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-hongkong-3","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-hongkong"}, {"Key":"Zone", "Value":"ap-hongkong-3"}]}'

# Southeast Asia Pacific (Singapore) : ap-singapore
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-singapore-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-singapore"}, {"Key":"Zone", "Value":"ap-singapore-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-singapore-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-singapore"}, {"Key":"Zone", "Value":"ap-singapore-2"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-singapore-3","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-singapore"}, {"Key":"Zone", "Value":"ap-singapore-3"}]}'

# Southeast Asia Pacific (Bangkok) : ap-bangkok
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-bangkok-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-bangkok"}, {"Key":"Zone", "Value":"ap-bangkok-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-bangkok-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-bangkok"}, {"Key":"Zone", "Value":"ap-bangkok-2"}]}'

# South Asia Pacific (Mumbai) : ap-mumbai
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-mumbai-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-mumbai"}, {"Key":"Zone", "Value":"ap-mumbai-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-mumbai-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-mumbai"}, {"Key":"Zone", "Value":"ap-mumbai-2"}]}'

# Northeast Asia Pacific (Seoul) : ap-seoul
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-seoul-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-seoul"}, {"Key":"Zone", "Value":"ap-seoul-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-seoul-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-seoul"}, {"Key":"Zone", "Value":"ap-seoul-2"}]}'

# Northeast Asia Pacific (Tokyo) : ap-tokyo
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-tokyo-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-tokyo"}, {"Key":"Zone", "Value":"ap-tokyo-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-tokyo-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"ap-tokyo"}, {"Key":"Zone", "Value":"ap-tokyo-2"}]}'

# Western U.S. (Silicon Valley) : na-siliconvalley
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-siliconvalley-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"na-siliconvalley"}, {"Key":"Zone", "Value":"na-siliconvalley-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-siliconvalley-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"na-siliconvalley"}, {"Key":"Zone", "Value":"na-siliconvalley-2"}]}'

# Eastern U.S. (Virginia) : na-ashburn
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-ashburn-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"na-ashburn"}, {"Key":"Zone", "Value":"na-ashburn-1"}]}'

curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-ashburn-2","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"na-ashburn"}, {"Key":"Zone", "Value":"na-ashburn-2"}]}'

# North America (Toronto) : na-toronto
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-toronto-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"na-toronto"}, {"Key":"Zone", "Value":"na-toronto-1"}]}'

# Europe (Frankfurt) : eu-frankfurt
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-frankfurt-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-frankfurt"}, {"Key":"Zone", "Value":"eu-frankfurt-1"}]}'

# Europe (Moscow) : eu-moscow
curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"tencent-moscow-1","ProviderName":"TENCENT", "KeyValueInfoList": [{"Key":"Region", "Value":"eu-moscow"}, {"Key":"Zone", "Value":"eu-moscow-1"}]}'


### Cloud Connection Config Info
# South China (Guangzhou) : ap-guangzhou  $$$(Caution!!) ap-guangzhou-5 zone은 없음.
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-guangzhou1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-guangzhou-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-guangzhou2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-guangzhou-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-guangzhou3-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-guangzhou-3"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-guangzhou4-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-guangzhou-4"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-guangzhou5-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-guangzhou-5"}'

# East China (Shanghai) : ap-shanghai
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-shanghai1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-shanghai-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-shanghai2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-shanghai-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-shanghai3-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-shanghai-3"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-shanghai4-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-shanghai-4"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-shanghai5-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-shanghai-5"}'


# East China (Nanjing) : ap-nanjing
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-nanjing1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-nanjing-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-nanjing2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-nanjing-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-nanjing3-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-nanjing-3"}'

# North China (Beijing) : ap-beijing
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-beijing1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-beijing-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-beijing2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-beijing-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-beijing3-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-beijing-3"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-beijing4-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-beijing-4"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-beijing5-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-beijing-5"}'

# Southwest China (Chengdu) : ap-chengdu
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-chengdu1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-chengdu-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-chengdu2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-chengdu-2"}'

# Southwest China (Chongqing) : ap-chongqing
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-chongqing1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-chongqing-1"}'

# Hong Kong/Macao/Taiwan (Hong Kong, China) : ap-hongkong
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-hongkong1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-hongkong-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-hongkong2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-hongkong-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-hongkong3-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-hongkong-3"}'

# Southeast Asia Pacific (Singapore) : ap-singapore
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-singapore1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-singapore-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-singapore2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-singapore-2"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-singapore3-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-singapore-3"}'

# Southeast Asia Pacific (Bangkok) : ap-bangkok
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-bangkok1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-bangkok-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-bangkok2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-bangkok-2"}'

# South Asia Pacific (Mumbai) : ap-mumbai
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-mumbai1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-mumbai-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-mumbai2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-mumbai-2"}'

# Northeast Asia Pacific (Seoul) : ap-seoul
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-seoul1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-seoul-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-seoul2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-seoul-2"}'

# Northeast Asia Pacific (Tokyo) : ap-tokyo
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-tokyo1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-tokyo-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-tokyo2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-tokyo-2"}'

# Western U.S. (Silicon Valley) : na-siliconvalley
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-siliconvalley1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-siliconvalley-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-siliconvalley2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-siliconvalley-2"}'

# Eastern U.S. (Virginia) : na-ashburn
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-ashburn1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-ashburn-1"}'

curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-ashburn2-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-ashburn-2"}'

# North America (Toronto) : na-toronto
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-toronto1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-toronto-1"}'

# Europe (Frankfurt) : eu-frankfurt
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-frankfurt1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-frankfurt-1"}'

# Europe (Moscow) : eu-moscow
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' -d '{"ConfigName":"tencent-moscow1-config","ProviderName":"TENCENT", "DriverName":"tencent-driver01", "CredentialName":"tencent-credential01", "RegionName":"tencent-moscow-1"}'
