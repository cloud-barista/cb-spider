VERSION := $(shell git describe --tags --abbrev=8 | sed 's/-g.*//')
COMMIT_SHA := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date)

export CGO_CFLAGS := -Wno-deprecated-declarations

default: swag
	@printf '\t[CB-Spider] building ./bin/cb-spider...\n'
	@go mod download
	@go mod tidy
	@CGO_ENABLED=1 go build -ldflags="-X 'main.Version=$(VERSION)' \
				-X 'main.CommitSHA=$(COMMIT_SHA)' \
				-X 'main.BuildTime=$(BUILD_TIME)'" \
			-o bin/cb-spider ./api-runtime

dyna plugin plug dynamic: swag
	@printf '\t[CB-Spider] building ./bin/cb-spider-dyna with plugin mode...\n'
	@go mod download
	@go mod tidy
	@CGO_ENABLED=1 go build -tags dyna -ldflags="-X 'main.Version=$(VERSION)' \
				-X 'main.CommitSHA=$(COMMIT_SHA)' \
				-X 'main.BuildTime=$(BUILD_TIME)'" \
			-o bin/cb-spider-dyna ./api-runtime
	@./build_all_driver_lib.sh;

cc: swag
	@printf '\t[CB-Spider] build ./bin/cb-spider-arm for linux arm...\n'
	GOOS=linux GOARCH=arm go build -o bin/cb-spider-arm ./api-runtime



SOURCES = ./api/docs.go ./api/swagger.json ./api/swagger.yaml

SED_REPLACE = \
	-e 's/github_com_cloud-barista_cb-spider_cloud-control-manager_cloud-driver_interfaces_resources/spider/g' \
	-e 's/restruntime/spider/g' \
	-e 's/github_com_cloud-barista_cb-spider_api-runtime_common-runtime/spider/g' \
	-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager_driver-info-manager/spider.cim/g' \
	-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager_credential-info-manager/spider.cim/g' \
	-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager_region-info-manager/spider.cim/g' \
	-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager_connection-config-info-manager/spider.cim/g' \
	-e 's/github_com_cloud-barista_cb-spider_cloud-info-manager/spider.cim/g'

swag swagger:
	@printf '\t[CB-Spider] generating Swagger documentations...\n'
	@~/go/bin/swag i -g api-runtime/rest-runtime/CBSpiderRuntime.go -d ./,./api-runtime/common-runtime,./cloud-control-manager,./cloud-info-manager,./info-store -o api > /dev/null
	@$(MAKE) --no-print-directory replace

replace:
	@for file in $(SOURCES); do \
		sed -i.bak $(SED_REPLACE) $$file; \
		rm -f $$file.bak; \
	done


clean clear:
	@printf '\t[CB-Spider] cleaning...\n'
	@rm -rf bin/cb-spider bin/cb-spider-dyna bin/cb-spider-arm bin/cb-spider-android-arm64



# =============================================================
# [Prerequisite Installation and Environment Setup for Android ARM64 Build]
#
# 1. Move to your home directory:
#    cd ~
#
# 2. Download and unzip Android NDK:
#    wget https://dl.google.com/android/repository/android-ndk-r26d-linux.zip
#    unzip android-ndk-r26d-linux.zip
#
# 3. Set environment variable (for bash):
#    export ANDROID_NDK_HOME=$$HOME/android-ndk-r26d
# =============================================================
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  NDK_PREBUILT := darwin-x86_64
else
  NDK_PREBUILT := linux-x86_64
endif

android-arm64: swag
	@printf '\t[CB-Spider] build ./bin/cb-spider-android-arm64 for Android arm64...\n'
	@PATH="$(ANDROID_NDK_HOME)/toolchains/llvm/prebuilt/$(NDK_PREBUILT)/bin:$$PATH" \
	CC=aarch64-linux-android21-clang \
	CGO_ENABLED=1 \
	GOOS=android GOARCH=arm64 \
	CGO_CFLAGS="--sysroot=$(ANDROID_NDK_HOME)/toolchains/llvm/prebuilt/$(NDK_PREBUILT)/sysroot" \
	CGO_LDFLAGS="--sysroot=$(ANDROID_NDK_HOME)/toolchains/llvm/prebuilt/$(NDK_PREBUILT)/sysroot" \
	go build -ldflags="-X 'main.Version=$(VERSION)' \
		-X 'main.CommitSHA=$(COMMIT_SHA)' \
		-X 'main.BuildTime=$(BUILD_TIME)'" \
		-o bin/cb-spider-android-arm64 ./api-runtime


