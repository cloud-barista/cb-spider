source setup.env

curl -sX GET "http://$RESTSERVER:1024/connectionconfig" |json_pp

