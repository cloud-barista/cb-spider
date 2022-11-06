#!/bin/bash

source ../driver-list.env

CMD=./4.key-delete.sh


function ClearAll() {
        for DRIVER in "${DRIVERS[@]}"
        do
                echo  ============ clear ${DRIVER} ... ============
                $CMD $DRIVER
        done
        }

ClearAll

echo -e "\n\n"

