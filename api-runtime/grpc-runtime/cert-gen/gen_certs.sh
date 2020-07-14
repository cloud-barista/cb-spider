#!/bin/bash

if [ ! -d "./certs" ]; then
  mkdir $CBSPIDER_ROOT/certs
fi

# Create Root signing Key
openssl genrsa -out $CBSPIDER_ROOT/certs/ca.key 4096

# Generate self-signed Root certificate
openssl req -new -x509 -key $CBSPIDER_ROOT/certs/ca.key -sha256 -subj "/C=KR/ST=DJ/O=Test CA, Inc." -days 3650 -out $CBSPIDER_ROOT/certs/ca.crt

# Create a Key certificate for your service
openssl genrsa -out $CBSPIDER_ROOT/certs/server.key 4096

# Create signing CSR
openssl req -new -key $CBSPIDER_ROOT/certs/server.key -out $CBSPIDER_ROOT/certs/server.csr -config certificate.conf

# Generate a certificate for the service
openssl x509 -req -in $CBSPIDER_ROOT/certs/server.csr -CA $CBSPIDER_ROOT/certs/ca.crt -CAkey $CBSPIDER_ROOT/certs/ca.key -CAcreateserial -out $CBSPIDER_ROOT/certs/server.crt -days 3650 -sha256 -extfile certificate.conf -extensions req_ext