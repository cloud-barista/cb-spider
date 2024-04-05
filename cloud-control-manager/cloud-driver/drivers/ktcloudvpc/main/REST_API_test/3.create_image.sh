#!/bin/bash
source ./1.export.env

echo "curl -v -s -X POST $OS_VOLUME_API/$PROJECT_ID/volumes/$VOLUME_ID/action"
curl -v -s -X POST $OS_VOLUME_API/$PROJECT_ID/volumes/$VOLUME_ID/action --header "X-Auth-Token: $OS_TOKEN" 'Content-Type: application/json' \
--data-raw '
{
	"os-volume_upload_image": {
		"force": true,
		"image_name": "new-image-1",
		"container_format": "bare",
		"disk_format": "qcow2"
	}
}' ; echo 

echo -e "\n"
