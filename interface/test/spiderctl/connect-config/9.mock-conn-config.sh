RESTSERVER=localhost

 # for Cloud Driver Info
 $CBSPIDER_ROOT/interface/spctl driver create --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{
      "DriverName":"mock-driver01",
      "ProviderName":"MOCK", 
      "DriverLibFileName":"mock-driver-v1.0.so"
    }'

 # for Cloud Credential Info
# for Mock
$CBSPIDER_ROOT/interface/spctl credential create --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{
      "CredentialName":"mock-credential01",
      "ProviderName":"MOCK",
      "KeyValueInfoList": [
        {"Key":"MockName", "Value":"mock_name00"}
      ]
    }'

 # Cloud Region Info
 $CBSPIDER_ROOT/interface/spctl region create --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{
      "RegionName":"mock-region01",
      "ProviderName":"MOCK",
      "KeyValueInfoList": [
        {"Key":"Region", "Value":"default"}
      ]
    }' 

 # Cloud Connection Config Info
 $CBSPIDER_ROOT/interface/spctl connection create --config $CBSPIDER_ROOT/interface/spctl.conf -i json -d \
    '{
      "ConfigName":"mock-config01",
      "ProviderName":"MOCK", 
      "DriverName":"mock-driver01", 
      "CredentialName":"mock-credential01", 
      "RegionName":"mock-region01"
    }'
