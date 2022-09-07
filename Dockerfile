##############################################################
## Stage 1 - Go Build
##############################################################

#FROM golang:alpine AS builder
FROM golang:1.19 AS builder

ENV GO111MODULE on

#RUN apk update && apk add --no-cache bash
#RUN apt update

#RUN apk add gcc

ADD . /go/src/github.com/cloud-barista/cb-spider

WORKDIR /go/src/github.com/cloud-barista/cb-spider

#RUN ./build_all_driver_lib.sh

WORKDIR api-runtime

RUN GOOS=linux go build -tags cb-spider -o cb-spider -v

#############################################################
## Stage 2 - Application Setup
##############################################################

FROM ubuntu:latest as prod

RUN apt update

RUN apt install -y ca-certificates

# use bash
RUN rm /bin/sh && ln -s /bin/bash /bin/sh

WORKDIR /root/go/src/github.com/cloud-barista/cb-spider

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/cloud-driver-libs/ /root/go/src/github.com/cloud-barista/cb-spider/cloud-driver-libs/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/conf/ /root/go/src/github.com/cloud-barista/cb-spider/conf/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/api-runtime/cb-spider /root/go/src/github.com/cloud-barista/cb-spider/api-runtime/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/admin-web/images/cb-spider-circle-logo.png /root/go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/admin-web/images/

#COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/setup.env /root/go/src/github.com/cloud-barista/cb-spider/
#RUN /bin/bash -c "source /root/go/src/github.com/cloud-barista/cb-spider/setup.env"
ENV CBSPIDER_ROOT /root/go/src/github.com/cloud-barista/cb-spider
ENV CBSTORE_ROOT /root/go/src/github.com/cloud-barista/cb-spider
ENV CBLOG_ROOT /root/go/src/github.com/cloud-barista/cb-spider
ENV PLUGIN_SW OFF
ENV MEERKAT OFF

ENTRYPOINT [ "/root/go/src/github.com/cloud-barista/cb-spider/api-runtime/cb-spider" ]

EXPOSE 1024
EXPOSE 2048
EXPOSE 4096
