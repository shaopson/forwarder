#!/bin/bash

VERSION=0
if [ -n "$1" ]; then
  VERSION=$1
else
  echo "missing version"
  exit
fi

mkdir -p dist
cd dist

# linux amd64
DEST=linux_amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $DEST/forwarder-server ../cmd/forwarder-server/main.go
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $DEST/forwarder-cli ../cmd/forwarder-cli/main.go
cp ../cmd/forwarder-server/forwarder.server.conf  $DEST
cp ../cmd/forwarder-cli/forwarder.client.conf  $DEST
tar -zcf forwarder_${VERSION}_${DEST}.tar.gz $DEST

# darwin  amd64
DEST=darwin_amd64
mkdir -p $DEST
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $DEST/forwarder-server ../cmd/forwarder-server/main.go
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $DEST/forwarder-cli ../cmd/forwarder-cli/main.go
cp ../cmd/forwarder-server/forwarder.server.conf  $DEST
cp ../cmd/forwarder-cli/forwarder.client.conf  $DEST
tar -zcf forwarder_${VERSION}_${DEST}.tar.gz $DEST

# darwin  arm64
DEST=darwin_arm64
mkdir -p $DEST
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $DEST/forwarder-server ../cmd/forwarder-server/main.go
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $DEST/forwarder-cli ../cmd/forwarder-cli/main.go
cp ../cmd/forwarder-server/forwarder.server.conf  $DEST
cp ../cmd/forwarder-cli/forwarder.client.conf  $DEST
tar -zcf forwarder_${VERSION}_${DEST}.tar.gz $DEST
