##############################################################
## Stage 1 - Go Build
##############################################################

FROM golang:1.25.0 AS builder

ENV GO111MODULE=on

# Install git for version information
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

#RUN apk update && apk add --no-cache bash
#RUN apt update

#RUN apk add gcc

ADD . /go/src/github.com/cloud-barista/cb-spider

WORKDIR /go/src/github.com/cloud-barista/cb-spider

# Note: Docker image provides static mode only (dynamic mode available in binary releases)
WORKDIR api-runtime

RUN VERSION=$(git describe --tags --abbrev=8 2>/dev/null | sed 's/-g.*//' || echo "unknown") && \
    COMMIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    BUILD_TIME=$(date) && \
    CGO_ENABLED=0 GOOS=linux go build -tags cb-spider \
    -ldflags="-X 'main.Version=${VERSION}' -X 'main.CommitSHA=${COMMIT_SHA}' -X 'main.BuildTime=${BUILD_TIME}'" \
    -o cb-spider -v

#############################################################
## Stage 2 - Application Setup
##############################################################

FROM ubuntu:22.04 AS prod

# Note: Combine update and install in a single RUN to avoid stale package list from cached layers.
# --no-install-recommends: Skip unnecessary recommended packages to reduce image size.
# rm -rf /var/lib/apt/lists/*: Clean up apt cache (does not affect installed packages).
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*

# use bash
RUN rm /bin/sh && ln -s /bin/bash /bin/sh

WORKDIR /root/go/src/github.com/cloud-barista/cb-spider

# Note: cloud-driver-libs not needed for static mode
COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/conf/ /root/go/src/github.com/cloud-barista/cb-spider/conf/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/api-runtime/cb-spider /root/go/src/github.com/cloud-barista/cb-spider/api-runtime/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/admin-web/images/ /root/go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/admin-web/images/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/admin-web/html/ /root/go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/admin-web/html/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/api/ /root/go/src/github.com/cloud-barista/cb-spider/api/

ENV CBSPIDER_ROOT=/root/go/src/github.com/cloud-barista/cb-spider
ENV CBLOG_ROOT=/root/go/src/github.com/cloud-barista/cb-spider
# Static mode only (dynamic mode available in binary releases)
ENV PLUGIN_SW=OFF

ENTRYPOINT [ "/root/go/src/github.com/cloud-barista/cb-spider/api-runtime/cb-spider" ]

EXPOSE 1024
