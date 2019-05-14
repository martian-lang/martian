#!/usr/bin/env bash
# Used for generating the bazel workspace status file

echo STABLE_MARTIAN_VERSION $(git describe --tags --always --dirty)
echo STABLE_MARTIAN_RELEASE false
