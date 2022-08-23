default: cli
		@echo -e '\t[CB-Spider] build ./bin/cb-spider....'
		@go mod download
		@go mod tidy
		@go build -o bin/cb-spider ./api-runtime
dyna plugin plug dynamic: cli
		@echo -e '\t[CB-Spider] build ./bin/cb-spider with plugin mode...'
		@go mod download
	        @go build -tags dyna -o bin/cb-spider-dyna ./api-runtime
		@./build_all_driver_lib.sh;
cc:
		@echo -e '\t[CB-Spider] build ./bin/cb-spider-arm for arm...'
	        GOOS=linux GOARCH=arm go build -o cb-spider-arm ./api-runtime
clean clear:
		@echo -e '\t[CB-Spider] cleaning...'
	        @rm -rf bin/cb-spider bin/cb-spider-dyna bin/cb-spider-arm
	        @rm -rf dist-tmp

cli-dist dist-cli: cli
		@echo -e '\t[CB-Spider] tar spctl... to dist'
		@mkdir -p /tmp/spider/dist/conf 
		@cp ./interface/spctl ./interface/spctl.conf /tmp/spider/dist 1> /dev/null
		@cp ./conf/log_conf.yaml /tmp/spider/dist/conf 1> /dev/null
		@mkdir -p ./dist
		@tar -zcvf ./dist/spctl-`(date +%Y.%m.%d.%H)`.tar.gz -C /tmp/spider/dist ./ 1> /dev/null
		@rm -rf /tmp/spider
cli:
		@echo -e '\t[CB-Spider] build ./interface/spctl...'
		@go mod download
		@go mod tidy
		@go build -ldflags="-X 'github.com/cloud-barista/cb-spider/interface/cli/spider/cmd.Version=v0.6.0' \
			-X 'github.com/cloud-barista/cb-spider/interface/cli/spider/cmd.CommitSHA=`(git rev-parse --short HEAD)`' \
			-X 'github.com/cloud-barista/cb-spider/interface/cli/spider/cmd.User=`(id -u -n)`' \
			-X 'github.com/cloud-barista/cb-spider/interface/cli/spider/cmd.Time=`(date)`'" \
			-o ./interface/spctl ./interface/cli/spider/spider.go
mini-dist dist-mini: dyna
		@echo -e '\t[CB-Spider] tar spider-mini... to dist'
		@mkdir -p /tmp/spider/dist
		@cp ./setup.env /tmp/spider/dist 1> /dev/null
		@mkdir -p /tmp/spider/dist/bin
		@cp ./bin/* /tmp/spider/dist/bin 1> /dev/null
		@mkdir -p /tmp/spider/dist/cloud-driver-libs
		@cp ./cloud-driver-libs/* /tmp/spider/dist/cloud-driver-libs 1> /dev/null
		@mkdir -p /tmp/spider/dist/conf
		@cp ./conf/* /tmp/spider/dist/conf 1> /dev/null
		@mkdir -p /tmp/spider/dist/api-runtime/rest-runtime/admin-web/images
		@cp ./api-runtime/rest-runtime/admin-web/images/cb-spider-circle-logo.png \
		      /tmp/spider/dist/api-runtime/rest-runtime/admin-web/images/cb-spider-circle-logo.png 1> /dev/null
		@mkdir -p ./dist
		@tar -zcvf ./dist/spider-mini-`(date +%Y.%m.%d.%H)`.tar.gz -C /tmp/spider/dist ./ 1> /dev/null
		@rm -rf /tmp/spider

swag swagger:
		@echo -e '\t[CB-Spider] build Swagger docs'
		@~/go/bin/swag i -g api-runtime/rest-runtime/CBSpiderRuntime.go -o api-runtime/rest-runtime/docs

