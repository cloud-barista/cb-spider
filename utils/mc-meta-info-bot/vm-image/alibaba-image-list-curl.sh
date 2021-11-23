#!/bin/bash

REGIONS=( alibaba-singapore-config alibaba-tokyo-config alibaba-beijing-config alibaba-ulanqab-config alibaba-london-config )

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
