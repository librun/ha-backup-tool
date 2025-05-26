#!/usr/bin/env bash

CURRENT_DIR=$(dirname "$(realpath $0)")
BUILD_DIR="${CURRENT_DIR}/build"
NAME="ha-backup-tool"
VERSION=${1}

echo "Application version ${VERSION}"

cleanup() {
    rm -rf ${BUILD_DIR}/*
}

function make_release() {
    local arch="$1"
    local os="$2"
    local release_name="$3"
    if [ -z "${arch}" ] || [ -z "${os}" ] || [ -z "${release_name}" ]; then
        echo "args are not set" >&2
        return 1
    fi
    local ext="$4"

    local dir="${BUILD_DIR}/${release_name}"

    mkdir -p "${dir}"
    env GOARCH="${arch}" GOOS="${os}" CGO_ENABLED=0 go build -ldflags "-s -w -X main.AppVersion=${VERSION}" -o "${dir}/${NAME}${ext}"

    cp LICENSE "${dir}"
    cp README.md "${dir}"
    #cp CHANGELOG.md "${dir}"

    cd ${BUILD_DIR}
    case "${os}" in
    linux | darwin)
        tar -zcvf "${release_name}.tar.gz" "${release_name}"
        md5sum "${release_name}.tar.gz" >>checksum.txt
        ;;
    windows)
        zip -r "${release_name}.zip" "${release_name}"
        md5sum "${release_name}.zip" >>checksum.txt
        ;;
    esac
    rm -r "${release_name}"
    cd ../
}

if [ -z "${VERSION}" ]; then
    echo "VERSION is not set. Use ./release.sh 0.0.0" >&2
    exit 1
fi

cleanup

touch ${BUILD_DIR}/checksum.txt

make_release 386 linux "${NAME}-linux-i386"
make_release amd64 linux "${NAME}-linux-amd64"
make_release arm64 linux "${NAME}-linux-arm64"

make_release 386 windows "${NAME}-win32" .exe
make_release amd64 windows "${NAME}-win64" .exe
make_release arm64 windows "${NAME}-win-arm64" .exe

make_release amd64 darwin "${NAME}-darwin-amd64"
make_release arm64 darwin "${NAME}-darwin-arm64"
