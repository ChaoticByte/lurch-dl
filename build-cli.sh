#!/usr/bin/env bash

setopt -e

WORKDIR="./cli"
CORE_DIR="./core"
OUTPUT_DIR="../dist"

function gobuild {
    printf -- "-> ${GOOS}\t${GOARCH}\t${OUTPUT_FILE}   "
    go build -ldflags="-X 'main.Version=${VERSION}' -X 'github.com/ChaoticByte/lurch-dl/core.Version=${CORE_VERSION}'" -o "${OUTPUT_DIR}/${OUTPUT_FILE}" && printf "\t✔\n"
}

read -r CORE_VERSION < "${CORE_DIR}/VERSION"

cd "${WORKDIR}"
read -r VERSION < ./VERSION

NAME_BASE="lurchdl-cli_${VERSION}_core${CORE_VERSION}"

echo "Building ${NAME_BASE} into ${OUTPUT_DIR}"

GOOS=windows GOARCH=386   OUTPUT_FILE=${NAME_BASE}_32bit.exe   gobuild
GOOS=windows GOARCH=amd64 OUTPUT_FILE=${NAME_BASE}_64bit.exe   gobuild
GOOS=windows GOARCH=arm64 OUTPUT_FILE=${NAME_BASE}_arm64.exe   gobuild
GOOS=linux   GOARCH=386   OUTPUT_FILE=${NAME_BASE}_linux_i386  gobuild
GOOS=linux   GOARCH=amd64 OUTPUT_FILE=${NAME_BASE}_linux_amd64 gobuild
GOOS=linux   GOARCH=arm   OUTPUT_FILE=${NAME_BASE}_linux_arm   gobuild
GOOS=linux   GOARCH=arm64 OUTPUT_FILE=${NAME_BASE}_linux_arm64 gobuild

cd ..

printf -- "Creating tag cli${VERSION}_core${CORE_VERSION}"
git tag -f "cli${VERSION}_core${CORE_VERSION}" && printf "\t\t✔\n"
