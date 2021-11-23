#!/bin/bash

REGIONS=( tencent-seoul1-config tencent-tokyo1-config tencent-beijing3-config tencent-guangzhou3-config tencent-frankfurt1-config )

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
