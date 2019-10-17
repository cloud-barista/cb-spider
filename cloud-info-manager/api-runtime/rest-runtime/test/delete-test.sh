RESTSERVER=node12

 # for Cloud Driver Info
curl -X DELETE http://$RESTSERVER:1024/connectionconfig/aws-config01
curl -X DELETE http://$RESTSERVER:1024/connectionconfig/azure-config01

curl -X DELETE http://$RESTSERVER:1024/driver/aws-driver01
curl -X DELETE http://$RESTSERVER:1024/driver/azure-driver01
curl -X DELETE http://$RESTSERVER:1024/driver/k8s-driver-V0.5

 # for Cloud Credential Info
curl -X DELETE http://$RESTSERVER:1024/credential/aws-credential01
curl -X DELETE http://$RESTSERVER:1024/credential/azure-credential01

 # for Cloud Region Info
curl -X DELETE http://$RESTSERVER:1024/region/aws-region01
curl -X DELETE http://$RESTSERVER:1024/region/azure-region01

 # for Cloud Connection Config Info
