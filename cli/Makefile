default:
	@echo -e '\t[CB-Spider] build spctl...'
	@go mod download
	@go mod tidy
	@cp ../api/swagger.json ./cmd
	@go build -ldflags="-X 'github.com/cloud-barista/cb-spider/cli/cmd.Version=v0.9.4' \
		-X 'github.com/cloud-barista/cb-spider/cli/cmd.CommitSHA=`(git rev-parse --short HEAD)`' \
		-X 'github.com/cloud-barista/cb-spider/cli/cmd.User=`(id -u -n)`' \
		-X 'github.com/cloud-barista/cb-spider/cli/cmd.Time=`(date)`'" \
		-o ./spctl spctl.go
	@rm ./cmd/swagger.json

clean clear:
	@echo -e '\t[CB-Spider] cleaning...'
	@rm -rf ./spctl
