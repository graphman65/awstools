#!/usr/bin/env bash
set -eu

base=$(dirname $0)/..
sums_file=${base}/bin/SHA256SUMS

mkdir -p ${base}/bin
rm -f ${base}/bin/*.asc
rm -f ${sums_file}*

version=$(cat ${base}/VERSION)
commit=$(git rev-parse --short HEAD)

mac_supported="iam-session kms-env aws-dump ecr-get-login ec2-describe-instances ec2-ip-from-name"

echo "Building ${version} (${commit})"
find ${base} -name "main.go" | while read src; do
    src=$(realpath --relative-to=${base} ${src})
    if [ $# -ge 1 ] && [ "$1" != "" ]; then
      if [ "$1" != "${src}" ]; then
        continue
      fi
    fi

    fname=$(echo ${src} | awk -F/ '{print $1"-"$2}')
    name=bin/${fname}
    echo "  ${name}"
    folder=`dirname ${src}`
    if [ ! -f ${folder}/Makefile ]; then
        CGO_ENABLED=0 go build -installsuffix cgo -o ${base}/${name} -ldflags="-s -w -X github.com/hamstah/awstools/common.Version=${version} -X github.com/hamstah/awstools/common.CommitHash=${commit}" ${folder}/*.go
        gpg --armor --detach-sig ${base}/${name}

        if [[ ${mac_supported} =~ ${fname} ]]; then
          GOOS=darwin CGO_ENABLED=0 go build -installsuffix cgo -o ${base}/${name}_darwin -ldflags="-s -w -X github.com/hamstah/awstools/common.Version=${version} -X github.com/hamstah/awstools/common.CommitHash=${commit}" ${folder}/*.go
          gpg --armor --detach-sig ${base}/${name}_darwin
        fi

    else
        cd ${folder}
        make
        cd -
    fi
done

cd ${base}/bin
find . -type f ! -name "*.asc"  | xargs sha256sum > SHA256SUMS
gpg --armor --detach-sig SHA256SUMS
cd -
