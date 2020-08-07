#!/bin/bash

# gogo proto compile
cd $CBSPIDER_ROOT/api-runtime/grpc-runtime/idl/gogoproto
protoc \
    gogo.proto \
		-I . \
		-I $GOPATH/src/github.com/gogo/protobuf/protobuf \
		-I $GOPATH/src \
		--gofast_out=plugins=grpc,paths=source_relative,\
Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:\
.

cp -rf gogo.pb.go $CBSPIDER_ROOT/api-runtime/grpc-runtime/stub/gogoproto
rm gogo.pb.go

# cbpider proto compile
cd $CBSPIDER_ROOT/api-runtime/grpc-runtime/idl/cbspider
protoc \
    cbspider.proto \
		-I . \
		-I $GOPATH/src/github.com/gogo/protobuf/protobuf \
		-I $GOPATH/src/github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/idl/gogoproto \
		--gofast_out=plugins=grpc:\
.	

cp -rf cbspider.pb.go $CBSPIDER_ROOT/api-runtime/grpc-runtime/stub/cbspider
rm cbspider.pb.go

