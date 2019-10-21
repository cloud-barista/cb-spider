RESTSERVER=node12

 # for Cloud Driver Info
curl -X GET http://$RESTSERVER:1024/driver |json_pp

 # for Cloud Credential Info
curl -X GET http://$RESTSERVER:1024/credential |json_pp

 # for Cloud Region Info
curl -X GET http://$RESTSERVER:1024/region |json_pp

 # for Cloud Connection Config Info
curl -X GET http://$RESTSERVER:1024/connectionconfig |json_pp
