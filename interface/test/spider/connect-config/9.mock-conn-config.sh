RESTSERVER=localhost

 # for Cloud Driver Info
 $CBSPIDER_ROOT/interface/spider driver create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{
      "DriverName":"mock-driver01",
      "ProviderName":"MOCK", 
      "DriverLibFileName":"mock-driver-v1.0.so"
    }'

 # for Cloud Credential Info
# for Mock
$CBSPIDER_ROOT/interface/spider credential create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{
      "CredentialName":"mock-credential01",
      "ProviderName":"MOCK",
      "KeyValueInfoList": [
        {"Key":"MockName", "Value":"mock_name00"}
      ]
    }'

 # Cloud Region Info
 $CBSPIDER_ROOT/interface/spider region create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{
      "RegionName":"mock-region01",
      "ProviderName":"MOCK",
      "KeyValueInfoList": [
        {"Key":"Region", "Value":"default"}
      ]
    }' 

 # Cloud Connection Config Info
 $CBSPIDER_ROOT/interface/spider connection create --config $CBSPIDER_ROOT/interface/grpc_conf.yaml -i json -d \
    '{
      "ConfigName":"mock-config01",
      "ProviderName":"MOCK", 
      "DriverName":"mock-driver01", 
      "CredentialName":"mock-credential01", 
      "RegionName":"mock-region01"
    }'
