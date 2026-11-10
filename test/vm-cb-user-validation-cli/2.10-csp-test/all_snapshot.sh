#!/bin/bash

CSPLIST=( aws azure gcp alibaba tencent ibm openstack ncp nhn kt )

function run() {
    param=$1
    num=0
    for CSP in "${CSPLIST[@]}"
    do
        echo "============ test ${CSP} ... ============"

	./one_csp_snapshot.sh ${CSP} ${param} &

        echo -e "\n\n"
    done
}

run "$@"

