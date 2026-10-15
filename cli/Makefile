VERSION := $(shell git describe --tags --abbrev=8 | sed 's/-g.*//')
COMMIT_SHA := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date)

default:
	@printf '\t[CB-Spider] building cli spctl...\n'
	@go mod download
	@go mod tidy
	@cp ../api/swagger.json ./cmd
	@go build -ldflags="-X 'github.com/cloud-barista/cb-spider/cli/cmd.Version=$(VERSION)' \
		-X 'github.com/cloud-barista/cb-spider/cli/cmd.CommitSHA=$(COMMIT_SHA)' \
		-X 'github.com/cloud-barista/cb-spider/cli/cmd.BuildTime=$(BUILD_TIME)'" \
		-o ./spctl spctl.go
	@rm ./cmd/swagger.json

clean clear:
	@printf '\t[CB-Spider] cleaning cli spctl...\n'
	@rm -rf ./spctl

