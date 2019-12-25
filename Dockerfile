##############################################################
## Stage 1 - Go Build
##############################################################

FROM golang:alpine AS builder

RUN apk update && apk add --no-cache bash

#RUN apk add gcc

ADD . /go/src/github.com/cloud-barista/cb-spider

WORKDIR /go/src/github.com/cloud-barista/cb-spider

#RUN ./build_all_driver_lib.sh

WORKDIR api-runtime/rest-runtime

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -extldflags "-static"' -tags cb-spider -o cb-spider -v

#############################################################
## Stage 2 - Application Setup
##############################################################

FROM ubuntu:latest

# use bash
RUN rm /bin/sh && ln -s /bin/bash /bin/sh

WORKDIR /app

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/cloud-driver-libs/* /app/cloud-driver-libs/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/conf/* /app/conf/

COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/cb-spider /app/api-runtime/rest-runtime/

#COPY --from=builder /go/src/github.com/cloud-barista/cb-spider/setup.env /app/
#RUN /bin/bash -c "source /app/setup.env"
ENV CBSPIDER_ROOT /app
ENV CBSTORE_ROOT /app
ENV CBLOG_ROOT /app

ENTRYPOINT [ "/app/api-runtime/rest-runtime/cb-spider" ]

EXPOSE 1024
