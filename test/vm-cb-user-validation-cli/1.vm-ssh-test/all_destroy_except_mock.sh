#!/bin/bash

CSPLIST=( aws azure gcp alibaba tencent ibm openstack ktcloud ktcloudvpc ncp ncpvpc nhncloud )

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
