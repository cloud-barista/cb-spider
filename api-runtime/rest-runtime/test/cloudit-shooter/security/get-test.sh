source ../setup.env

SECURITY_ID=CB-SecGroup
curl -sX GET http://$RESTSERVER:1024/securitygroup/$SECURITY_ID?connection_name=cloudit-config01 | json_pp &
