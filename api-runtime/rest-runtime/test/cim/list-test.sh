RESTSERVER=node12

 # for Cloud Driver Info
curl -X GET http://$RESTSERVER:1024/spider/driver |json_pp

 # for Cloud Credential Info
curl -X GET http://$RESTSERVER:1024/spider/credential |json_pp

 # for Cloud Region Info
curl -X GET http://$RESTSERVER:1024/spider/region |json_pp

 # for Cloud Connection Config Info
curl -X GET http://$RESTSERVER:1024/spider/connectionconfig |json_pp
