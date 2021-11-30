#!/bin/bash

SLEEP=1

function loop() {
	while true
        do
		clear
		echo -e "Key #: `redis-cli dbsize | sed 's/(integer)//g'`"

                sleep $SLEEP
        done
        }

loop
