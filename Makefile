VERSION := $(shell git describe --tags --abbrev=8 | sed 's/-g.*//')
COMMIT_SHA := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date)

default: swag
	@echo -e '\t[CB-Spider] building ./bin/cb-spider...'
	@go mod download
	@go mod tidy
	@go build -ldflags="-X 'main.Version=$(VERSION)' \
	                    -X 'main.CommitSHA=$(COMMIT_SHA)' \
	                    -X 'main.BuildTime=$(BUILD_TIME)'" \
			-o bin/cb-spider ./api-runtime

dyna plugin plug dynamic: swag
	@echo -e '\t[CB-Spider] building ./bin/cb-spider-dyna with plugin mode...'
	@go mod download
	@go mod tidy
	@go build -tags dyna -ldflags="-X 'main.Version=$(VERSION)' \
                            -X 'main.CommitSHA=$(COMMIT_SHA)' \
                            -X 'main.BuildTime=$(BUILD_TIME)'" \
			-o bin/cb-spider-dyna ./api-runtime
	@./build_all_driver_lib.sh;

cc: swag
		@echo -e '\t[CB-Spider] build ./bin/cb-spider-arm for arm...'
	        GOOS=linux GOARCH=arm go build -o cb-spider-arm ./api-runtime

clean clear:
		@echo -e '\t[CB-Spider] cleaning...'
	        @rm -rf bin/cb-spider bin/cb-spider-dyna bin/cb-spider-arm

swag swagger:
	@echo -e '\t[CB-Spider] generating Swagger documentations...'
	@~/go/bin/swag i -g api-runtime/rest-runtime/CBSpiderRuntime.go -d ./,./api-runtime/common-runtime,./cloud-control-manager,./cloud-info-manager,./info-store -o api > /dev/null
	@sed -i -e 's/github_com_cloud-barista_cb-spider_cloud-control-manager_cloud-driver_interfaces_resources/spider/g' \
			-e 's/restruntime/spider/g' \
			-e 's/github_com_cloud-barista_cb-spider_api-runtime_common-runtime/spider/g' \
			-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager_driver-info-manager/spider.cim/g' \
			-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager_credential-info-manager/spider.cim/g' \
			-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager_region-info-manager/spider.cim/g' \
			-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager_connection-config-info-manager/spider.cim/g' \
			-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager/spider.cim/g' \
			./api/docs.go ./api/swagger.json ./api/swagger.yaml
