#!/bin/bash
API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD


RESTSERVER=localhost

while true
do
        curl -u $API_USERNAME:$API_PASSWORD -sX GET http://$RESTSERVER:1024/spider/splockinfo -H 'Content-Type: application/json' |json_pp
        sleep 1
        #clear
done
