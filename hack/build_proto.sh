#!/bin/bash
set -e

OS="$(go env GOHOSTOS)"
ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null 2>&1 && pwd & )"

mkdir -p ${ROOT}/proto

bazel run //:gazelle
bazel build //_proto:go_default_library

cp ${ROOT}/bazel-out/${OS}-fastbuild/bin/_proto/proto_go_proto_/github.com/joesonw/drlee/proto/* ${ROOT}/proto/.
