#!/bin/bash

if [ -n "$(gofmt -l src/martian)" ]; then
    echo "Go code is not formatted:"
    gofmt -d src/martian
    exit 1
fi
