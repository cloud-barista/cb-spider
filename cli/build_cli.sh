#!/bin/bash

cp ../api/swagger.json ./cmd
go mod tidy
go build 
rm ./cmd/swagger.json
