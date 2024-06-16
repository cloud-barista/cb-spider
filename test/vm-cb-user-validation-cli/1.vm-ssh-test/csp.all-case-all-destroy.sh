#!/bin/bash

CSPLIST=( mock mock2 mock3 mock4 mock5 aws azure gcp alibaba tencent ibm openstack ktcloud ncpvpc )

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
