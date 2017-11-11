#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui"
fi
PATH=../../bin:$PATH
touch fail1
mrp pipeline.mro pipeline_test
mrp pipeline.mro pipeline_test
