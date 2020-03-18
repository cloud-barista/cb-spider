RESTSERVER=node12

 # for Cloud Driver Info
curl -X DELETE http://$RESTSERVER:1024/spider/connectionconfig/aws-config01
curl -X DELETE http://$RESTSERVER:1024/spider/connectionconfig/azure-config01

curl -X DELETE http://$RESTSERVER:1024/spider/driver/aws-driver01
curl -X DELETE http://$RESTSERVER:1024/spider/driver/azure-driver01
curl -X DELETE http://$RESTSERVER:1024/spider/driver/k8s-driver-V0.5

 # for Cloud Credential Info
curl -X DELETE http://$RESTSERVER:1024/spider/credential/aws-credential01
curl -X DELETE http://$RESTSERVER:1024/spider/credential/azure-credential01

 # for Cloud Region Info
curl -X DELETE http://$RESTSERVER:1024/spider/region/aws-region01
curl -X DELETE http://$RESTSERVER:1024/spider/region/azure-region01

 # for Cloud Connection Config Info
