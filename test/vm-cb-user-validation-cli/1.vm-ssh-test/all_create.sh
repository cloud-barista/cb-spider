#!/bin/bash

CSPLIST=( mock aws azure gcp alibaba tencent ibm openstack ktclassic ktcloudvpc ncp nhn )

function run() {
    param=$1
    num=0
    for CSP in "${CSPLIST[@]}"
    do
        echo "============ test ${CSP} ... ============"

	./one_csp_run.sh ${CSP} ${param} &

        echo -e "\n\n"
    done
}

run "$@"

