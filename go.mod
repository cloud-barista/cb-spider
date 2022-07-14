module github.com/cloud-barista/cb-spider

go 1.16

replace (
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.3
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/docker/distribution => github.com/docker/distribution v2.8.0+incompatible
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2
	//google.golang.org/api => google.golang.org/api v0.19.0
	google.golang.org/api => google.golang.org/api v0.16.0
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0
)

require (
	github.com/Azure/azure-sdk-for-go v55.6.0+incompatible
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.0.1 // indirect
	github.com/IBM/go-sdk-core/v5 v5.5.1
	github.com/IBM/vpc-go-sdk v0.8.0
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.1662
	github.com/aws/aws-sdk-go v1.39.4
	github.com/bramvdbogaerde/go-scp v1.0.0
	github.com/chyeh/pubip v0.0.0-20170203095919-b7e679cf541c
	github.com/cloud-barista/cb-log v0.5.0
	github.com/cloud-barista/cb-store v0.5.2
	github.com/containerd/containerd v1.6.6 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-resty/resty/v2 v2.6.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang-jwt/jwt/v4 v4.0.0
	github.com/golang/protobuf v1.5.2
	github.com/gophercloud/gophercloud v0.18.0
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/labstack/echo/v4 v4.3.0
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/opentracing/opentracing-go v1.2.0
	github.com/prometheus/client_golang v1.11.1
	github.com/rs/xid v1.3.0
	github.com/sirupsen/logrus v1.8.1
	github.com/snowzach/rotatefilehook v0.0.0-20180327172521-2f64f265f58c
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/swaggo/echo-swagger v1.1.0
	github.com/swaggo/swag v1.7.0
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb v1.0.415
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.0.415
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm v1.0.206
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc v1.0.206
	github.com/uber/jaeger-client-go v2.29.1+incompatible
	github.com/uber/jaeger-lib v2.4.0+incompatible // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5
	google.golang.org/api v0.81.0
	google.golang.org/grpc v1.46.2
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b

)

retract (
	v0.3.12
	v0.3.11
)
