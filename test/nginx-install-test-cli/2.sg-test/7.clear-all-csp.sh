#!/bin/bash

source ../driver-list.env

CMD=./6.sg-delete.sh


function ClearAll() {
        for DRIVER in "${DRIVERS[@]}"
        do
                echo  ============ clear ${DRIVER} ... ============
                $CMD $DRIVER
        done
        }

ClearAll

echo -e "\n\n"

