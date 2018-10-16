#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui"
fi
PATH=../../bin:$PATH
mkdir -p ar_fail
export FAILFILE_DIR=$PWD/ar_fail
touch $FAILFILE_DIR/fail1
mrp --autoretry=1 --psdir=ar_pipeline_fail pipeline.mro pipeline_fail
