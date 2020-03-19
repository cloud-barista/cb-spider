##############################################################
## Stage 1 - Go Build
##############################################################

#FROM golang:alpine AS builder
FROM golang:1.12.4 AS builder

ENV GO111MODULE on

#RUN apk update && apk add --no-cache bash
#RUN apt update

#RUN apk add gcc

ADD . /go/src/github.com/cloud-barista/cb-spider

WORKDIR /go/src/github.com/cloud-barista/cb-spider

RUN ./build_all_driver_lib.sh

WORKDIR api-runtime/rest-runtime

RUN GOOS=linux go build -tags cb-spider -o cb-spider -v

#############################################################
## Stage 2 - Application Setup
##############################################################

FROM ubuntu:latest

RUN apt update

RUN apt install -y ca-certificates

# use bash
RUN rm /bin/sh && ln -s /bin/bash /bin/sh

WORKDIR /root/go/src/github.com/cloud-barista/cb-spider

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/cloud-driver-libs/* /root/go/src/github.com/cloud-barista/cb-spider/cloud-driver-libs/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/conf/* /root/go/src/github.com/cloud-barista/cb-spider/conf/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/cb-spider /root/go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/

#COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/setup.env /root/go/src/github.com/cloud-barista/cb-spider/
#RUN /bin/bash -c "source /root/go/src/github.com/cloud-barista/cb-spider/setup.env"
ENV CBSPIDER_ROOT /root/go/src/github.com/cloud-barista/cb-spider
ENV CBSTORE_ROOT /root/go/src/github.com/cloud-barista/cb-spider
ENV CBLOG_ROOT /root/go/src/github.com/cloud-barista/cb-spider

ENTRYPOINT [ "/root/go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/cb-spider" ]

EXPOSE 1024
