#!/bin/bash

#REGIONS=( aws-seoul-config )
#REGIONS=( aws-ohio-config AWS:US-EAST-2:US-EAST-2A )
REGIONS=( aws-seoul-config aws-tokyo-config aws-ohio-config aws-oregon-config aws-paris-config )

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
