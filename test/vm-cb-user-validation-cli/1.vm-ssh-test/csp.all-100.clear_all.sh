#!/bin/bash

CSPLIST=( mock mock2 mock3 mock4 mock5 aws azure gcp alibaba ncpvpc )

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

