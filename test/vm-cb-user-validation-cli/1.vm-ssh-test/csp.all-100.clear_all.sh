#!/bin/bash

CSPLIST=( mock aws azure gcp alibaba tencent ibm openstack ktcloud ktcloudvpc ncp ncpvpc nhncloud )

function run() {
    param=$1
    for CSP in "${CSPLIST[@]}"
    do
        echo "============ test ${CSP} ... ============"

        ./100.clear_all.sh ${CSP} ${param} &

        echo -e "\n\n"
    done
}

run "$@"

