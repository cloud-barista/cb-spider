#!/bin/bash

echo "####################################################################"
echo "## Cloud Driver Info"
echo "####################################################################"

# Load sensitive information from .openstack-credential
OPENSTACK_CREDENTIAL_FILE="./.openstack-credential"
if [[ -f $OPENSTACK_CREDENTIAL_FILE ]]; then
    source $OPENSTACK_CREDENTIAL_FILE
else
    echo "Error: Credential file not found at $OPENSTACK_CREDENTIAL_FILE"
    exit 1
fi

if [[ -z "$openstack_identity_endpoint" || -z "$openstack_username" || -z "$openstack_password" || -z "$openstack_domain_name" || -z "$openstack_project_id" ]]; then
    echo "Error: Missing one or more required OpenStack credential variables in credential file."
    exit 1
fi

# Cloud Driver Info
curl -X POST http://$RESTSERVER:1024/spider/driver \
    -H 'Content-Type: application/json' \
    -d '{
        "DriverName":"openstack-driver01",
        "ProviderName":"OPENSTACK", 
        "DriverLibFileName":"openstack-driver-v1.0.so"
    }'

echo "####################################################################"
echo "## Cloud Credential Info"
echo "####################################################################"

# Cloud Credential Info
curl -X POST http://$RESTSERVER:1024/spider/credential \
    -H 'Content-Type: application/json' \
    -d '{
        "CredentialName":"openstack-credential01", 
        "ProviderName":"OPENSTACK", 
        "KeyValueInfoList": [
            {"Key":"IdentityEndpoint", "Value":"'"$openstack_identity_endpoint"'"},
            {"Key":"Username", "Value":"'"$openstack_username"'"},
            {"Key":"Password", "Value":"'"$openstack_password"'"},
            {"Key":"DomainName", "Value":"'"$openstack_domain_name"'"},
            {"Key":"ProjectID", "Value":"'"$openstack_project_id"'"}
        ]
    }'

echo "####################################################################"
echo "## Cloud Region Info"
echo "####################################################################"

# Cloud Region Info
curl -X POST http://$RESTSERVER:1024/spider/region \
    -H 'Content-Type: application/json' \
    -d '{
        "RegionName":"openstack-region01",
        "ProviderName":"OPENSTACK",
        "KeyValueInfoList": [
            {"Key":"Region", "Value":"RegionOne"}
        ]
    }'

echo "####################################################################"
echo "## Cloud Connection Config Info"
echo "####################################################################"

# Cloud Connection Config Info
curl -X POST http://$RESTSERVER:1024/spider/connectionconfig \
    -H 'Content-Type: application/json' \
    -d '{
        "ConfigName":"openstack-config01",
        "ProviderName":"OPENSTACK", 
        "DriverName":"openstack-driver01", 
        "CredentialName":"openstack-credential01", 
        "RegionName":"openstack-region01"
    }'
