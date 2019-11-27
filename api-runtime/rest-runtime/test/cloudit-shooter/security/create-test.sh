source ../setup.env


SECURITY_ID=CB-SecGroup
curl -sX POST http://$RESTSERVER:1024/securitygroup?connection_name=cloudit-config01 -H 'Content-Type: application/json' -d '{ 
	"Name": "'${SECURITY_ID}'", 
	"SecurityRules": [{"FromPort": "22", "ToPort" : "22", "IPProtocol" : "tcp", "Direction" : "inbound"}] 
}' |json_pp &
