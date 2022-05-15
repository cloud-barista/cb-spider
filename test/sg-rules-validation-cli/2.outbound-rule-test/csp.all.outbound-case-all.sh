#!/bin/bash


CSPLIST=( aws azure gcp alibaba tencent ibm openstack cloudit )
CSPLIST=( aws azure gcp alibaba tencent ibm )

function run() {
        num=0
        for CSP in "${CSPLIST[@]}"
        do
                echo  ============ test ${CSP} ... ============

                if [ "${CSP}" = "azure" ]; then
                        export SLEEP=60
                fi
                if [ "${CSP}" = "gcp" ]; then
                        export SLEEP=60
                fi

		./00.prepare-00.sh ${CSP}
		./all.outbound-case-all.sh ${CSP}
		./100.clear_all.sh ${CSP}


                if [ "${CSP}" = "azure" ]; then
                        unset SLEEP
                fi
                if [ "${CSP}" = "gcp" ]; then
                        unset SLEEP
                fi

                num=`expr $num + 1`

		echo -e "\n\n"
        done
        }

run

