#!/bin/sh
# Builds the plugin binary for every published platform.
#
#   VERSION=<version> BUILD=<build> sh scripts/build.sh

set -eu

export CGO_ENABLED=0
ldflags="-X main.version=${VERSION:-dev} -X main.build=${BUILD:-local}"

set -x

GOOS=linux   GOARCH=amd64 go build -ldflags "$ldflags" -o release/linux/amd64/drone-maven .
GOOS=linux   GOARCH=arm64 go build -ldflags "$ldflags" -o release/linux/arm64/drone-maven .
GOOS=windows GOARCH=amd64 go build -ldflags "$ldflags" -o release/windows/amd64/drone-maven.exe .
