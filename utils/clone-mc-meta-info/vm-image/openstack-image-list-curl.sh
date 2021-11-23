#!/bin/bash

REGIONS=( openstack-config01 openstack-config01 )

function listImage() {
        for REGION in "${REGIONS[@]}"
        do
                echo  ============ List Image ${REGION} ... ============
                /bin/bash -c './image-list-curl.sh '$REGION'' || return 1
        done
        }

listImage
