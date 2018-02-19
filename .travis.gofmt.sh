#!/bin/bash

if [ -n "$(gofmt -l martian cmd)" ]; then
    echo "Go code is not formatted:"
    gofmt -d martian cmd
    exit 1
fi
