#!/usr/bin/env bash
set -o pipefail
bazel build //tools/docs
cp "$(bazel info bazel-bin)/"tools/docs/*.md "$(bazel info workspace)/"tools/docs/