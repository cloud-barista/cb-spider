#!/bin/bash


CSPLIST=( aws azure gcp alibaba tencent ibm openstack cloudit )
CSPLIST=( aws azure alibaba tencent )

function run() {
        num=0
        for CSP in "${CSPLIST[@]}"
        do
                echo  ============ test ${CSP} ... ============

		if [ "${CSP}" = "azure" ]; then
			export SLEEP=80
		fi

		./00.prepare-00.sh ${CSP}
		./all.inbound-case-all.sh ${CSP}
		./100.clear_all.sh ${CSP}

                if [ "${CSP}" = "azure" ]; then
                        unset SLEEP
                fi

                num=`expr $num + 1`

		echo -e "\n\n"
        done
        }

run

