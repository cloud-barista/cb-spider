API_USERNAME=${API_USERNAME:-admin}
API_PASSWORD=$API_PASSWORD

RESTSERVER=localhost


#==================== Connection

# Connection for SEOUL
curl -u $API_USERNAME:$API_PASSWORD -sX DELETE http://$RESTSERVER:1024/spider/connectionconfig/CICS-CRMS-AWS-SEOUL -H 'Content-Type: application/json' | json_pp

# Connection for TOKYO
curl -u $API_USERNAME:$API_PASSWORD -sX DELETE http://$RESTSERVER:1024/spider/connectionconfig/CICS-CRMS-AWS-TOKYO -H 'Content-Type: application/json' | json_pp

# Connection for MUMBAI
curl -u $API_USERNAME:$API_PASSWORD -sX DELETE http://$RESTSERVER:1024/spider/connectionconfig/CICS-CRMS-AWS-MUMBAI -H 'Content-Type: application/json' | json_pp



#==================== Region

# for Seoul Region
curl -u $API_USERNAME:$API_PASSWORD -sX DELETE http://$RESTSERVER:1024/spider/region/aws-seoul -H 'Content-Type: application/json' | json_pp

# for Tokyo Region
curl -u $API_USERNAME:$API_PASSWORD -sX DELETE http://$RESTSERVER:1024/spider/region/aws-tokyo -H 'Content-Type: application/json' | json_pp

# for Mumbai Region
curl -u $API_USERNAME:$API_PASSWORD -sX DELETE http://$RESTSERVER:1024/spider/region/aws-mumbai -H 'Content-Type: application/json' | json_pp



#==================== Credential

 # for Cloud Credential Info
curl -u $API_USERNAME:$API_PASSWORD -sX DELETE http://$RESTSERVER:1024/spider/credential/aws-credential01 -H 'Content-Type: application/json' | json_pp



#==================== Driver

 # for Cloud Driver Info
curl -u $API_USERNAME:$API_PASSWORD -sX DELETE http://$RESTSERVER:1024/spider/driver/aws-driver01 -H 'Content-Type: application/json' | json_pp


