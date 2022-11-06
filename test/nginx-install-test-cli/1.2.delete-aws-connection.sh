RESTSERVER=localhost


#==================== Connection

# Connection for SEOUL
curl -sX DELETE http://$RESTSERVER:1024/spider/connectionconfig/CICS-CRMS-AWS-SEOUL -H 'Content-Type: application/json' | json_pp

# Connection for TOKYO
curl -sX DELETE http://$RESTSERVER:1024/spider/connectionconfig/CICS-CRMS-AWS-TOKYO -H 'Content-Type: application/json' | json_pp

# Connection for MUMBAI
curl -sX DELETE http://$RESTSERVER:1024/spider/connectionconfig/CICS-CRMS-AWS-MUMBAI -H 'Content-Type: application/json' | json_pp



#==================== Region

# for Seoul Region
curl -sX DELETE http://$RESTSERVER:1024/spider/region/aws-seoul -H 'Content-Type: application/json' | json_pp

# for Tokyo Region
curl -sX DELETE http://$RESTSERVER:1024/spider/region/aws-tokyo -H 'Content-Type: application/json' | json_pp

# for Mumbai Region
curl -sX DELETE http://$RESTSERVER:1024/spider/region/aws-mumbai -H 'Content-Type: application/json' | json_pp



#==================== Credential

 # for Cloud Credential Info
curl -sX DELETE http://$RESTSERVER:1024/spider/credential/aws-credential01 -H 'Content-Type: application/json' | json_pp



#==================== Driver

 # for Cloud Driver Info
curl -sX DELETE http://$RESTSERVER:1024/spider/driver/aws-driver01 -H 'Content-Type: application/json' | json_pp


