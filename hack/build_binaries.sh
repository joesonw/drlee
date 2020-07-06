#!/bin/bash -e

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null 2>&1 && pwd & )"

rm -rf ${ROOT}/binaries
mkdir -p ${ROOT}/binaries

ALL_OS=(darwin linux windows)
ALL_ARCH=(amd64)
BAZEL=${BAZEL_BIN:=bazel}
HOSTOS="$(go env GOHOSTOS)"

for OS in ${ALL_OS[@]}; do
    EXT=""
    if [[ $OS == "windows" ]]; then
        EXT=".exe"
    fi
    for ARCH in ${ALL_ARCH[@]}; do
          FILE="${ROOT}/binaries/drlee-${OS}-${ARCH}${EXT}"
          ${BAZEL} build --platforms=@io_bazel_rules_go//go/toolchain:${OS}_${ARCH} //:drlee
          cp ${ROOT}/bazel-out/${HOSTOS}-fastbuild/bin/drlee_/drlee${EXT} ${FILE}
    done
done


