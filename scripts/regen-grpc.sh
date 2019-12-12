#!/bin/bash

cwd=$(dirname "$(realpath $0)")
localbin=$(readlink -f $cwd/../.bin)

# check that protoc compiler exists and download it if required
PROTOBUF_VERSION=3.11.1
PROTOC_FILENAME=protoc-${PROTOBUF_VERSION}-linux-x86_64.zip
PROTOC_PATH=$localbin/protoc-$PROTOBUF_VERSION
if [ ! -d $PROTOC_PATH ] ; then
    mkdir -p $PROTOC_PATH
    curl -L https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOBUF_VERSION}/${PROTOC_FILENAME} > $localbin/$PROTOC_FILENAME
    mkdir -p $PROTOC_PATH
    unzip -o $localbin/$PROTOC_FILENAME -d $localbin/protoc-$PROTOBUF_VERSION
    rm $localbin/$PROTOC_FILENAME
fi

# it gets the version of protoc-gen-go from go.mod file
GOBIN=$localbin go install github.com/golang/protobuf/protoc-gen-go/

GOOGLE_PROTO_DIR=$PROTOC_PATH/include/google/protobuf

$PROTOC_PATH/bin/protoc --go_out=plugins=grpc,paths=source_relative:api -I$GOOGLE_PROTO_DIR:"$cwd/../api" $cwd/../api/*.proto --plugin=protoc-gen-go=$localbin/protoc-gen-go
