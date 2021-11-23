#!/bin/bash

REGIONS=( gcp-seoul-config gcp-tokyo-config gcp-iowa-config gcp-oregon-config gcp-london-config ) 

function listImage() {
        num=0
        for REGION in "${REGIONS[@]}"
        do
                echo  ============ List Image ${REGION} ... ============
                /bin/bash -c './image-list-curl.sh '$REGION'' || return 1

                num=`expr $num + 1`
        done
        }

listImage
