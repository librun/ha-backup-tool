#!/bin/bash

CURRENT_DIR=$(dirname "$(realpath $0)")
BUILD_DIR="${CURRENT_DIR}/build"
VERSION=$(cat ./main.go | grep -i -E "AppVersion\s+=" | cut -d'=' -f2 | tr -d '"' | tr -d '[:space:]')

echo "Application version ${VERSION}"

cleanup() {
    rm -rf ${BUILD_DIR}/*
}

build() {
    local OS=${1}
    local ARCH=${2}
    local EXT=${3}
    local OUTPUT="${BUILD_DIR}/ha-backup-tool_v${VERSION}_${OS}_${ARCH}${EXT}"
    echo "🪚  Building for ${OS} ${ARCH} ..."
    GOOS=${OS} GOARCH=${ARCH} go build -o ${OUTPUT} main.go

    gzip ${OUTPUT}
}

cleanup

build windows amd64 .exe
build windows 386 .exe
build windows arm64 .exe

build linux amd64
build linux 386
build linux arm64

build darwin amd64
build darwin arm64
