#!/bin/bash

RESTSERVER=3.36.93.11
RESTSERVER=localhost

while true
do
        curl -sX GET http://$RESTSERVER:1024/spider/splockinfo -H 'Content-Type: application/json' |json_pp
        sleep 1
        #clear
done
