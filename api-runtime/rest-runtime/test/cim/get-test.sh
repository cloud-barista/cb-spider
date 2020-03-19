RESTSERVER=node12

 # for Cloud Driver Info
curl -X GET http://$RESTSERVER:1024/spider/driver/aws-driver01 |json_pp
curl -X GET http://$RESTSERVER:1024/spider/driver/azure-driver01 |json_pp

 # for Cloud Credential Info
curl -X GET http://$RESTSERVER:1024/spider/credential/aws-credential01 |json_pp
curl -X GET http://$RESTSERVER:1024/spider/credential/azure-credential01 |json_pp

 # for Cloud Region Info
curl -X GET http://$RESTSERVER:1024/spider/region/aws-region01 |json_pp
curl -X GET http://$RESTSERVER:1024/spider/region/azure-region01 |json_pp

 # for Cloud Connection Config Info
curl -X GET http://$RESTSERVER:1024/spider/connectionconfig/aws-config01 |json_pp
curl -X GET http://$RESTSERVER:1024/spider/connectionconfig/azure-config01 |json_pp
