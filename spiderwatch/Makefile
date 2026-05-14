# SpiderWatch Makefile

APP        := spiderwatch
MODULE     := github.com/cloud-barista/cb-spiderwatch
CMD_DIR    := ./cmd/spiderwatch
BIN_DIR    := bin
BINARY     := $(BIN_DIR)/$(APP)
IMAGE_NAME := cloudbaristaorg/cb-spiderwatch
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

LDFLAGS    := -ldflags "-X main.version=$(VERSION)"

# Status Board
SB_APP     := statusboard
SB_CMD_DIR := ./cmd/statusboard
SB_BINARY  := $(BIN_DIR)/$(SB_APP)
SB_DIST    := dist/spider-statusboard

# Cross-compile targets for Status Board (GOOS/GOARCH overridable)
SB_GOOS   ?= $(shell go env GOOS)
SB_GOARCH ?= $(shell go env GOARCH)

.PHONY: all build run start stop clean docker-build docker-push tidy lint test \
        sb-build sb-build-linux sb-build-darwin sb-run sb-start sb-stop sb-dist sb-dist-linux

all: build

## build: compile the spiderwatch binary
build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BINARY) $(CMD_DIR)
	@echo "Built: $(BINARY)"

## run: build and run spiderwatch in the background (logs → terminal + logs/spiderwatch.log)
run: build
	@mkdir -p logs
	@./$(BINARY) -config conf/spiderwatch.yaml 2>&1 | tee -a logs/spiderwatch.log &
	@echo "SpiderWatch started — logs: logs/spiderwatch.log"

## start: alias for run
start: run

## stop: stop a running spiderwatch process (started with 'make run')
stop:
	@pkill -f "$(BINARY)" 2>/dev/null && echo "SpiderWatch stopped." || echo "SpiderWatch is not running."
	@rm -f .spiderwatch.pid

## tidy: tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

## lint: run go vet
lint:
	go vet ./...

## test: run unit tests
test:
	go test -v -count=1 ./...

## docker-build: build the Docker image
docker-build:
	docker build -t $(IMAGE_NAME):$(VERSION) -t $(IMAGE_NAME):latest .

## docker-push: push the Docker image to Docker Hub
docker-push: docker-build
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest

## clean: remove build artifacts
clean:
	rm -rf $(BIN_DIR) dist/

# ── Spider Status Board ──────────────────────────────────────────────────────

## sb-build: compile the statusboard binary (native platform)
sb-build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(SB_BINARY) $(SB_CMD_DIR)
	@echo "Built: $(SB_BINARY)"

## sb-build-linux: cross-compile the statusboard binary for Linux/amd64
sb-build-linux:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(SB_BINARY)-linux-amd64 $(SB_CMD_DIR)
	@echo "Built: $(SB_BINARY)-linux-amd64"

## sb-build-darwin: cross-compile the statusboard binary for macOS (amd64 + arm64)
sb-build-darwin:
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(SB_BINARY)-darwin-amd64 $(SB_CMD_DIR)
	@echo "Built: $(SB_BINARY)-darwin-amd64"
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(SB_BINARY)-darwin-arm64 $(SB_CMD_DIR)
	@echo "Built: $(SB_BINARY)-darwin-arm64"

## sb-run: build and run statusboard in the background (logs → statusboard.log)
sb-run: sb-build
	@./$(SB_BINARY) -config conf/statusboard.yaml 2>&1 | tee -a statusboard.log &
	@echo "Spider Status Board started — logs: statusboard.log"

## sb-start: alias for sb-run
sb-start: sb-run

## sb-stop: stop a running statusboard process
sb-stop:
	@pkill -f "$(SB_BINARY)" 2>/dev/null && echo "Spider Status Board stopped." || echo "Spider Status Board is not running."

## sb-dist: create self-contained deployment archives for all platforms
##          Produces:
##            dist/statusboard-<VERSION>-linux-amd64.tar.gz
##            dist/statusboard-<VERSION>-darwin-amd64.tar.gz
##            dist/statusboard-<VERSION>-darwin-arm64.tar.gz
sb-dist: sb-build-linux sb-build-darwin
	@echo "Creating Status Board distribution packages..."
	@mkdir -p dist
	@for PLATFORM in linux-amd64 darwin-amd64 darwin-arm64; do \
		rm -rf $(SB_DIST); \
		mkdir -p $(SB_DIST)/bin $(SB_DIST)/conf $(SB_DIST)/data/results $(SB_DIST)/logs; \
		cp $(SB_BINARY)-$$PLATFORM $(SB_DIST)/bin/$(SB_APP); \
		cp -r web/ $(SB_DIST)/web/; \
		cp conf/statusboard.yaml $(SB_DIST)/conf/statusboard.yaml; \
		printf '# Spider Status Board Makefile\n\nBIN  := bin/statusboard\n\n.PHONY: run start stop\n\nrun start:\n\t@./$$(BIN) -config conf/statusboard.yaml 2>&1 | tee -a statusboard.log &\n\t@echo "Spider Status Board started"\n\nstop:\n\t@pkill -f "./$$(BIN)" 2>/dev/null && echo "Stopped." || echo "Not running."\n' > $(SB_DIST)/Makefile; \
		tar -czf dist/statusboard-$(VERSION)-$$PLATFORM.tar.gz -C dist spider-statusboard; \
		echo "Distribution: dist/statusboard-$(VERSION)-$$PLATFORM.tar.gz"; \
	done
	@rm -rf $(SB_DIST)

## sb-dist-linux: build and package Linux/amd64 only
sb-dist-linux: sb-build-linux
	@echo "Creating Status Board distribution package (linux/amd64)..."
	@rm -rf $(SB_DIST)
	@mkdir -p $(SB_DIST)/bin $(SB_DIST)/conf $(SB_DIST)/data/results $(SB_DIST)/logs
	@cp $(SB_BINARY)-linux-amd64 $(SB_DIST)/bin/$(SB_APP)
	@cp -r web/ $(SB_DIST)/web/
	@cp conf/statusboard.yaml $(SB_DIST)/conf/statusboard.yaml
	@printf '# Spider Status Board Makefile\n\nBIN  := bin/statusboard\n\n.PHONY: run start stop\n\nrun start:\n\t@./$$(BIN) -config conf/statusboard.yaml 2>&1 | tee -a statusboard.log &\n\t@echo "Spider Status Board started"\n\nstop:\n\t@pkill -f "./$$(BIN)" 2>/dev/null && echo "Stopped." || echo "Not running."\n' > $(SB_DIST)/Makefile
	@tar -czf dist/statusboard-$(VERSION)-linux-amd64.tar.gz -C dist spider-statusboard
	@rm -rf $(SB_DIST)
	@echo "Distribution: dist/statusboard-$(VERSION)-linux-amd64.tar.gz"

## help: show this help
help:
	@grep -E '^##' Makefile | sed 's/## /  /'
