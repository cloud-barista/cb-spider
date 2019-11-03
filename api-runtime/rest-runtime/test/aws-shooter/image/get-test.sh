source ../setup.env

curl -X GET http://$RESTSERVER:1024/vmimage/${OHIO_IMG_ID1}?connection_name=aws-ohio-config |json_pp
curl -X GET http://$RESTSERVER:1024/vmimage/${OREGON_IMG_ID1}?connection_name=aws-oregon-config |json_pp
curl -X GET http://$RESTSERVER:1024/vmimage/${SINGAPORE_IMG_ID1}?connection_name=aws-singapore-config |json_pp
curl -X GET http://$RESTSERVER:1024/vmimage/${PARIS_IMG_ID1}?connection_name=aws-paris-config |json_pp
curl -X GET http://$RESTSERVER:1024/vmimage/${SAOPAULO_IMG_ID1}?connection_name=aws-saopaulo-config |json_pp

curl -X GET http://$RESTSERVER:1024/vmimage/${TOKYO_IMG_ID1}?connection_name=aws-tokyo-config |json_pp
