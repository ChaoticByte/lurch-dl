#!/usr/bin/env bash

WORKDIR="./cli"
CORE_DIR="./core"
OUTPUT_DIR="../dist"

function gobuild {
    printf -- "-> ${GOOS}\t${GOARCH}\t${OUTPUT_FILE}   "
    go build -ldflags="-X 'github.com/ChaoticByte/lurch-dl/core.Version=${VERSION}'" -o "${OUTPUT_DIR}/${OUTPUT_FILE}" && printf "\t✔\n"
}

read -r VERSION < ./VERSION

cd "${WORKDIR}"

NAME_BASE="lurch-dl_v${VERSION}"

echo "Building ${NAME_BASE} into ${OUTPUT_DIR}"

GOOS=linux   GOARCH=386   OUTPUT_FILE=${NAME_BASE}_linux_i386  gobuild
GOOS=linux   GOARCH=amd64 OUTPUT_FILE=${NAME_BASE}_linux_amd64 gobuild
GOOS=linux   GOARCH=arm   OUTPUT_FILE=${NAME_BASE}_linux_arm   gobuild
GOOS=linux   GOARCH=arm64 OUTPUT_FILE=${NAME_BASE}_linux_arm64 gobuild

cd ..

printf -- "Creating version tag"
git tag -f "v${VERSION}" -m "" && printf "\t\t✔\n"
