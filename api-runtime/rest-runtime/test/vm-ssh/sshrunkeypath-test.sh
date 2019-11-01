RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/sshrunkeypath -H 'Content-Type: application/json' -d '{
    "UserName": "ubuntu",
    "KeyPath"  : "/home/sean/.ssh/keyfile.pem",
    "ServerPort": "18.163.97.170:22",
    "Command": "/bin/hostname"
}'