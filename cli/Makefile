VERSION := $(shell git describe --tags --abbrev=8 | sed 's/-g.*//')
COMMIT_SHA := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date)

default:
	@echo "VERSION: $(VERSION)"
	@echo "COMMIT_SHA: $(COMMIT_SHA)"
	@echo "BUILD_TIME: $(BUILD_TIME)"
	@echo -e '\t[CB-Spider] build spctl...'
	@go mod download
	@go mod tidy
	@cp ../api/swagger.json ./cmd
	@go build -ldflags="-X 'github.com/cloud-barista/cb-spider/cli/cmd.Version=$(VERSION)' \
		-X 'github.com/cloud-barista/cb-spider/cli/cmd.CommitSHA=$(COMMIT_SHA)' \
		-X 'github.com/cloud-barista/cb-spider/cli/cmd.BuildTime=$(BUILD_TIME)'" \
		-o ./spctl spctl.go
	@rm ./cmd/swagger.json

clean clear:
	@echo -e '\t[CB-Spider] cleaning...'
	@rm -rf ./spctl

