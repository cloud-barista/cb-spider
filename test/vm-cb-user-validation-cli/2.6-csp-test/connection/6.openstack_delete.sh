#!/bin/bash

echo "####################################################################"
echo "## Cloud Connection Config Info Deletion"
echo "####################################################################"

# Cloud Connection Config Info Deletion
curl -X DELETE http://$RESTSERVER:1024/spider/connectionconfig/openstack-config01 \
    -H 'Content-Type: application/json'

echo "####################################################################"
echo "## Cloud Region Info Deletion"
echo "####################################################################"

# Cloud Region Info Deletion
curl -X DELETE http://$RESTSERVER:1024/spider/region/openstack-region01 \
    -H 'Content-Type: application/json'

echo "####################################################################"
echo "## Cloud Credential Info Deletion"
echo "####################################################################"

# Cloud Credential Info Deletion
curl -X DELETE http://$RESTSERVER:1024/spider/credential/openstack-credential01 \
    -H 'Content-Type: application/json'

echo "####################################################################"
echo "## Cloud Driver Info Deletion"
echo "####################################################################"

# Cloud Driver Info Deletion
curl -X DELETE http://$RESTSERVER:1024/spider/driver/openstack-driver01 \
    -H 'Content-Type: application/json'

