RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/sshcopykeypath -H 'Content-Type: application/json' -d '{
    "UserName": "ubuntu",
    "KeyPath"  : "/home/sean/.ssh/keyfile.pem",
    "ServerPort": "18.163.97.170:22",
    "SourceFile": "/home/sean/README.txt",
    "TargetFile": "/home/ubuntu/temp/README.txt"
}'