#!/usr/bin/env bash

set -e

WORKDIR="./cli"
CORE_DIR="./core"
OUTPUT_DIR="../dist"

VERSION=$(git describe --tags)

function gobuild {
    printf -- "-> ${GOOS}\t${GOARCH}\t${OUTPUT_FILE}"
    go build -ldflags="-X 'main.Version=${VERSION}'" -o "${OUTPUT_DIR}/${OUTPUT_FILE}"
    echo ""
}

cd "${WORKDIR}"

NAME_BASE="${VERSION}/lurch-dl"

echo "Building ${NAME_BASE} into ${OUTPUT_DIR}"

GOOS=linux GOARCH=386   OUTPUT_FILE=${NAME_BASE}_linux_i386  gobuild
GOOS=linux GOARCH=amd64 OUTPUT_FILE=${NAME_BASE}_linux_amd64 gobuild
GOOS=linux GOARCH=arm   OUTPUT_FILE=${NAME_BASE}_linux_arm   gobuild
GOOS=linux GOARCH=arm64 OUTPUT_FILE=${NAME_BASE}_linux_arm64 gobuild

cd ..
