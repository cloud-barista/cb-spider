default:
		@echo -e '\t[CB-Spider] build ./bin/cb-spider....'
		@go build -o bin/cb-spider ./api-runtime
dyna plugin plug dynamic:
		@echo -e '\t[CB-Spider] build ./bin/cb-spider with plugin mode...'
	        @go build -tags dyna -o bin/cb-spider-dyna ./api-runtime
		@./build_all_driver_lib.sh;
cc:
		@echo -e '\t[CB-Spider] build ./bin/cb-spider-arm for arm...'
	        GOOS=linux GOARCH=arm go build -o cb-spider-arm ./api-runtime
clean clear:
		@echo -e '\t[CB-Spider] cleaning...'
	        @rm -rf bin/cb-spider bin/cb-spider-dyna bin/cb-spider-arm
