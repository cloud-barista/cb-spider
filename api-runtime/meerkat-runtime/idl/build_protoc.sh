# Build IDL between MomKat and ChildKat.
# The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
# The CB-Spider Mission is to connect all the clouds with a single interface.
#
#      * Cloud-Barista: https://github.com/cloud-barista
#
# by CB-Spider Team, 2020.11.


#protoc -I grpc_def/ grpc_def/meerkat.proto --go_out=plugins=grpc:grpc_def
#protoc -I grpc_def/ grpc_def/meerkat.proto --go_out=plugins=grpc:$HOME/go/src
protoc -I ./ meerkat.proto --gofast_out=plugins=grpc:$HOME/go/src
