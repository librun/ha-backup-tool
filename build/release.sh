#!/usr/bin/env bash

CURRENT_DIR=$(dirname "$(realpath $0)")
BUILD_PATH="."
NAME="ha-backup-tool"
export VERSION=$(cat ${BUILD_PATH}/../main.go | grep -i -E "AppVersion\s+=" | cut -d'=' -f2 | tr -d '"' | tr -d '[:space:]')

echo "Application version ${VERSION}"
echo "Build path ${CURRENT_DIR}"

cleanup() {
    rm -f ${BUILD_PATH}/checksum*.txt
    rm -f ${BUILD_PATH}/ha-backup-tool*.tar.gz
    rm -f ${BUILD_PATH}/ha-backup-tool*.zip
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

    local dir="${BUILD_PATH}/${release_name}"

    mkdir -p "${dir}"
    env GOARCH="${arch}" GOOS="${os}" CGO_ENABLED=0 go build -o "${dir}/${NAME}${ext}" ../

    cp ${BUILD_PATH}/../LICENSE "${dir}"
    cp ${BUILD_PATH}/../README.md "${dir}"
    #cp ${BUILD_PATH}/../CHANGELOG.md "${dir}"

    case "${os}" in
    linux | darwin)
        tar -zcvf "${release_name}.tar.gz" "${release_name}"
        md5sum "${release_name}.tar.gz" >>checksum-md5.txt
        sha256sum "${release_name}.tar.gz" >>checksum-sha256.txt
        ;;
    windows)
        zip -r "${release_name}.zip" "${release_name}"
        md5sum "${release_name}.zip" >>checksum-md5.txt
        sha256sum "${release_name}.zip" >>checksum-sha256.txt
        ;;
    esac
    rm -r "${dir}"
}

cleanup

touch ${BUILD_PATH}/checksum-md5.txt
touch ${BUILD_PATH}/checksum-sha256.txt

make_release 386 linux "${NAME}-linux-i386"
make_release amd64 linux "${NAME}-linux-amd64"
make_release arm64 linux "${NAME}-linux-arm64"

make_release 386 windows "${NAME}-win32" .exe
make_release amd64 windows "${NAME}-win64" .exe
make_release arm64 windows "${NAME}-win-arm64" .exe

make_release amd64 darwin "${NAME}-darwin-amd64"
make_release arm64 darwin "${NAME}-darwin-arm64"

echo "############################"
echo "## Completed successfully ##"
echo "############################"
