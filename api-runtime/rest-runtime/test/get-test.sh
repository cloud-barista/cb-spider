RESTSERVER=node12

 # for Cloud Driver Info
curl -X GET http://$RESTSERVER:1024/driver/aws-driver01 |json_pp
curl -X GET http://$RESTSERVER:1024/driver/azure-driver01 |json_pp

 # for Cloud Credential Info
curl -X GET http://$RESTSERVER:1024/credential/aws-credential01 |json_pp
curl -X GET http://$RESTSERVER:1024/credential/azure-credential01 |json_pp

 # for Cloud Region Info
curl -X GET http://$RESTSERVER:1024/region/aws-region01 |json_pp
curl -X GET http://$RESTSERVER:1024/region/azure-region01 |json_pp

 # for Cloud Connection Config Info
curl -X GET http://$RESTSERVER:1024/connectionconfig/aws-config01 |json_pp
curl -X GET http://$RESTSERVER:1024/connectionconfig/azure-config01 |json_pp
