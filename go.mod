module github.com/cloud-barista/cb-spider

go 1.21

replace (
	github.com/IBM/vpc-go-sdk/0.23.0 => github.com/IBM/vpc-go-sdk v0.23.0
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible
)

require (
	github.com/Azure/azure-sdk-for-go v66.0.0+incompatible
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/IBM/go-sdk-core/v5 v5.10.2
	github.com/IBM/vpc-go-sdk v0.8.0
	github.com/IBM/vpc-go-sdk/0.23.0 v0.23.0
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.1743
	github.com/aws/aws-sdk-go v1.39.4
	github.com/bramvdbogaerde/go-scp v1.0.0
	github.com/chyeh/pubip v0.0.0-20170203095919-b7e679cf541c
	github.com/cloud-barista/cb-log v0.8.0
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/docker v24.0.7+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-resty/resty/v2 v2.6.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt/v4 v4.0.0
	github.com/golang/protobuf v1.5.3
	github.com/gophercloud/gophercloud v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/opentracing/opentracing-go v1.2.0
	github.com/prometheus/client_golang v1.11.1
	github.com/rs/xid v1.3.0
	github.com/sirupsen/logrus v1.9.0
	github.com/snowzach/rotatefilehook v0.0.0-20220211133110-53752135082d
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/swaggo/echo-swagger v1.1.0
	github.com/swaggo/swag v1.7.0
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb v1.0.415
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.0.493
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm v1.0.493
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc v1.0.206
	github.com/uber/jaeger-client-go v2.29.1+incompatible
	golang.org/x/crypto v0.17.0
	golang.org/x/oauth2 v0.7.0
	google.golang.org/api v0.114.0
	google.golang.org/grpc v1.56.3
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/IBM/platform-services-go-sdk v0.30.0
	github.com/NaverCloudPlatform/ncloud-sdk-go-v2 v1.6.6
	github.com/cloud-barista/ktcloud-sdk-go v0.2.1-0.20240123114820-8684cfc5deeb
	github.com/cloud-barista/nhncloud-sdk-go v0.0.0-20231114133737-f29e6fddb736
	github.com/go-openapi/strfmt v0.21.3
	github.com/hashicorp/go-version v1.6.0
	github.com/jeremywohl/flatten v1.0.1
	github.com/labstack/echo/v4 v4.9.0
	github.com/tencentcloud/tencentcloud-sdk-go-intl-en v3.0.531+incompatible
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs v1.0.492
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
)

require (
	github.com/itchyny/timefmt-go v0.1.5 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.17 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
)

require (
	cloud.google.com/go/compute v1.19.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.13 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.2 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.0.1 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200907205600-7a23bdc65eef // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/go-openapi/errors v0.20.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/spec v0.20.0 // indirect
	github.com/go-openapi/swag v0.19.12 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.7.1 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/itchyny/gojq v0.12.14
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/labstack/gommon v0.3.1 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.3-0.20211202183452-c5a74bcca799 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/swaggo/files v0.0.0-20190704085106-630677cd5c14 // indirect
	github.com/uber/jaeger-lib v2.4.0+incompatible // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	go.mongodb.org/mongo-driver v1.10.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/term v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11 // indirect
	golang.org/x/tools v0.6.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.66.2 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gorm.io/driver/sqlite v1.5.2
	gorm.io/gorm v1.25.2
	gotest.tools/v3 v3.0.3 // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

retract (
	v0.3.12
	v0.3.11
)
