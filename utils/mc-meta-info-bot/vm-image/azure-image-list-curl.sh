#!/bin/bash

REGIONS=( azure-koreacentral-config azure-japanwest-config azure-eastus-config azure-westus-config azure-northeu-config )

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
