#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui"
fi
PATH=../../bin:$PATH
mkdir -p retry
export FAILFILE_DIR=$PWD/retry
touch $FAILFILE_DIR/fail1
mrp --psdir=retry_pipeline_test pipeline.mro pipeline_test
mrp --psdir=retry_pipeline_test pipeline.mro pipeline_test
