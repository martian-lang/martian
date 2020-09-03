#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui --strict=error"
fi
PATH=../../bin:$PATH
mrp pipeline.mro pipeline_test
