module github.com/cloud-barista/cb-spider

go 1.25.0

replace (
	github.com/alibabacloud-go/ecs-20140526/v4 => github.com/alibabacloud-go/ecs-20140526/v4 v4.0.1
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible
)

require (
	github.com/IBM/go-sdk-core/v5 v5.19.0
	github.com/IBM/vpc-go-sdk v0.68.0
	github.com/aliyun/alibaba-cloud-sdk-go v1.63.103
	github.com/aws/aws-sdk-go v1.39.4
	github.com/bramvdbogaerde/go-scp v1.0.0
	github.com/cloud-barista/cb-log v0.9.1
	github.com/davecgh/go-spew v1.1.1
	github.com/fsnotify/fsnotify v1.7.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/gophercloud/gophercloud v1.3.0
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b // indirect
	github.com/rs/xid v1.6.0
	github.com/sirupsen/logrus v1.9.3
	github.com/snowzach/rotatefilehook v0.0.0-20220211133110-53752135082d
	github.com/spf13/cobra v1.2.1
	github.com/swaggo/swag v1.16.3
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb v1.0.415
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.1.8
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm v1.0.1064
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc v1.0.206
	golang.org/x/crypto v0.45.0
	golang.org/x/oauth2 v0.29.0
	google.golang.org/api v0.229.0
	google.golang.org/grpc v1.71.1 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go/filestore v1.10.2
	cloud.google.com/go/monitoring v1.24.2
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.1
	github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6 v6.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6 v6.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6 v6.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.8.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription v1.2.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/IBM/platform-services-go-sdk v0.30.0
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/NaverCloudPlatform/ncloud-sdk-go-v2 v1.6.27
	github.com/alibabacloud-go/cs-20151215/v4 v4.5.8
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.6
	github.com/alibabacloud-go/ecs-20140526/v4 v4.24.17
	github.com/alibabacloud-go/tea v1.3.7
	github.com/alibabacloud-go/vpc-20160428/v6 v6.4.0
	github.com/cloud-barista/ktcloud-sdk-go v0.2.1-0.20251103105234-6cbc5279b7fb
	github.com/cloud-barista/ktcloudvpc-sdk-go v0.0.0-20251103122025-f2d7abfe3716
	github.com/cloud-barista/nhncloud-sdk-go v0.0.2-0.20251103104208-d7c8b254bbda
	github.com/go-openapi/strfmt v0.23.0
	github.com/goccy/go-json v0.10.5
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/go-version v1.6.0
	github.com/itchyny/gojq v0.12.17
	github.com/jeremywohl/flatten v1.0.1
	github.com/labstack/echo/v4 v4.13.3
	github.com/minio/minio-go/v7 v7.0.94
	github.com/olekukonko/tablewriter v0.0.5
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/swaggo/echo-swagger v1.4.1
	github.com/tencentcloud/tencentcloud-sdk-go-intl-en v3.0.531+incompatible
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs v1.0.492
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cfs v1.1.8
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag v1.0.964
	golang.org/x/mod v0.29.0
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
)

require (
	cloud.google.com/go v0.118.3 // indirect
	cloud.google.com/go/auth v0.16.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/longrunning v0.6.6 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.2 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/endpoint-util v1.1.0 // indirect
	github.com/alibabacloud-go/openapi-util v0.1.1 // indirect
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7 // indirect
	github.com/aliyun/credentials-go v1.4.5 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-playground/validator/v10 v10.24.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/itchyny/timefmt-go v0.1.6 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/minio/crc64nvme v1.0.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240916144458-20a13a1f6b7c // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/swaggo/files/v2 v2.0.0 // indirect
	github.com/tinylib/msgp v1.3.0 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	google.golang.org/genproto v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250414145226-207652e42e2e // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250414145226-207652e42e2e // indirect
)

require (
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/spec v0.20.4 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/labstack/gommon v0.4.2
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	go.mongodb.org/mongo-driver v1.17.2 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/term v0.37.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gorm.io/driver/sqlite v1.5.5
	gorm.io/gorm v1.25.7
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

retract (
	v0.3.12
	v0.3.11
)
