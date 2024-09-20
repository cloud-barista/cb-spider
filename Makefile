default: swag
		@echo -e '\t[CB-Spider] build ./bin/cb-spider....'
		@go mod download
		@go mod tidy
		@go build -o bin/cb-spider ./api-runtime
dyna plugin plug dynamic: swag
		@echo -e '\t[CB-Spider] build ./bin/cb-spider with plugin mode...'
		@go mod download
	        @go build -tags dyna -o bin/cb-spider-dyna ./api-runtime
		@./build_all_driver_lib.sh;
cc: swag
		@echo -e '\t[CB-Spider] build ./bin/cb-spider-arm for arm...'
	        GOOS=linux GOARCH=arm go build -o cb-spider-arm ./api-runtime
clean clear:
		@echo -e '\t[CB-Spider] cleaning...'
	        @rm -rf bin/cb-spider bin/cb-spider-dyna bin/cb-spider-arm

swag swagger:
	@echo -e '\t[CB-Spider] generating Swagger documentation'
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
