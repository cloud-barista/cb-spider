#!/bin/bash

CSPLIST=( mock mock2 mock3 mock4 mock5 aws azure gcp alibaba ncpvpc )
CSPLIST=( ncpvpc )
CSPLIST=( mock )
CSPLIST=( aws )
CSPLIST=( ncpvpc )

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

