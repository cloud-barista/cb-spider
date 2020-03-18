source setup.env

curl -sX GET "http://$RESTSERVER:1024/spider/connectionconfig" |json_pp

