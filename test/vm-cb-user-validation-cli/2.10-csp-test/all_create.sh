#!/bin/bash
# usage: ./all_create.sh [number-of-VMs per CSP, default: 1]

CSPLIST=( aws azure gcp alibaba tencent ibm openstack ncp nhn kt )

function run() {
    param=${1:-1}   # number of VMs per CSP (default: 1)
    num=0
    for CSP in "${CSPLIST[@]}"
    do
        echo "============ test ${CSP} ... ============"

	./one_csp_run.sh ${CSP} ${param} &

        echo -e "\n\n"
    done
}

run "$@"

