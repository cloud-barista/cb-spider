#!/bin/bash

M1=60 	# 1 Min
M10=600 # 10 Min
M30=1800 # 30 Min
H1=3600  # 1 Hour
H3=10800 # 3 Hour
H6=21600 # 6 Hour
H12=43200 # 12 Hour
H24=86400 # 24 Hour


CNT=10
SLEEP=$M10

function loop() {
        for (( i=0; i < $CNT; i++ ))
        do
                echo  ============ List Image ${REGION} ... ============

		./gcp-image-list-curl.sh

                sleep $SLEEP
        done
        }

loop
