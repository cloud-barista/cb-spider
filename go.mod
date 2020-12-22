module github.com/cloud-barista/cb-spider

go 1.15

replace (
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.3
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)

require (
	cloud.google.com/go/pubsub v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go v49.1.0+incompatible
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.5
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.0.1 // indirect
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.785
	github.com/aws/aws-sdk-go v1.36.12
	github.com/bramvdbogaerde/go-scp v0.0.0-20200820121624-ded9ee94aef5
	github.com/chyeh/pubip v0.0.0-20170203095919-b7e679cf541c
	github.com/cloud-barista/cb-log v0.3.0-espresso
	github.com/cloud-barista/cb-store v0.3.0-espresso
	github.com/containerd/containerd v1.4.3 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.0.0-20200309214505-aa6a9891b09c
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.4 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/google/martian/v3 v3.0.0 // indirect
	github.com/google/pprof v0.0.0-20200708004538-1a94d8640e99 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.3.0 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/prometheus/client_golang v1.9.0
	github.com/rackspace/gophercloud v1.0.1-0.20161013232434-e00690e87603
	github.com/sirupsen/logrus v1.7.0
	github.com/snowzach/rotatefilehook v0.0.0-20180327172521-2f64f265f58c
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/uber/jaeger-lib v2.4.0+incompatible // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	go.opencensus.io v0.22.4 // indirect
	golang.org/x/crypto v0.0.0-20201217014255-9d1352758620
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20201216054612-986b41b23924
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/tools v0.0.0-20200825202427-b303f430e36d // indirect
	google.golang.org/api v0.15.0
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/grpc v1.33.0
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	gotest.tools/v3 v3.0.3 // indirect
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
	rsc.io/quote/v3 v3.1.0 // indirect
)
