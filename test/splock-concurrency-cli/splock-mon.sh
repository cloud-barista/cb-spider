#!/bin/bash
SPIDER_USERNAME=${SPIDER_USERNAME:-admin}
SPIDER_PASSWORD=$SPIDER_PASSWORD


RESTSERVER=localhost

while true
do
        curl -u $SPIDER_USERNAME:$SPIDER_PASSWORD -sX GET http://$RESTSERVER:1024/spider/splockinfo -H 'Content-Type: application/json' |json_pp
        sleep 1
        #clear
done
