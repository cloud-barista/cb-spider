RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/sshcopykeypath -H 'Content-Type: application/json' -d '{
    "UserName": "ubuntu",
    "KeyPath"  : "/home/ubuntu/.ssh/keyfile.pem",
    "ServerPort": "18.163.00.000:22",
    "SourceFile": "/home/ubuntu/README.txt",
    "TargetFile": "/home/ubuntu/temp/README.txt"
}'