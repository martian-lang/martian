#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui"
fi
PATH=../../bin:$PATH
mkdir -p fail
export FAILFILE_DIR=$PWD/fail
touch $FAILFILE_DIR/fail1
mrp pipeline.mro pipeline_fail
