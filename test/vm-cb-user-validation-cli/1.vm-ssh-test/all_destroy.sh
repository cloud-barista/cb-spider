#!/bin/bash

CSPLIST=( mock aws azure gcp alibaba tencent ibm openstack ktclassic ktcloudvpc ncp nhn )

function run() {
    param=$1
    for CSP in "${CSPLIST[@]}"
    do
        echo "============ test ${CSP} ... ============"

        ./10.destroy_all.sh ${CSP} &

        echo -e "\n\n"
    done
}

run "$@"
