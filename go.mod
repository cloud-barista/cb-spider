module github.com/cloud-barista/cb-spider

go 1.13

require (
	github.com/Azure/azure-sdk-for-go v37.2.0+incompatible
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.60.315
	github.com/aws/aws-sdk-go v1.27.0
	github.com/bramvdbogaerde/go-scp v0.0.0-20191005185035-c96fe084709e
	github.com/cloud-barista/cb-log v0.0.0-20191204151250-4d68d1150b4e
	github.com/cloud-barista/cb-store v0.0.0-20200103024537-5a9953f36101
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.18+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-00010101000000-000000000000 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.1 // indirect
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/prometheus/client_golang v1.3.0 // indirect
	github.com/racker/perigee v0.1.0 // indirect
	github.com/rackspace/gophercloud v1.0.0
	github.com/sirupsen/logrus v1.4.2
	github.com/snowzach/rotatefilehook v0.0.0-20180327172521-2f64f265f58c // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/valyala/fasttemplate v1.1.0 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/xujiajun/nutsdb v0.5.0 // indirect
	go.etcd.io/bbolt v1.3.3 // indirect
	go.etcd.io/etcd v3.3.18+incompatible // indirect
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/crypto v0.0.0-20191227163750-53104e6ec876
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	google.golang.org/api v0.15.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20191120175047-4206685974f2
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
