#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui"
fi
PATH=../../bin:$PATH
mkdir -p ar_pass
export FAILFILE_DIR=$PWD/ar_pass
echo "1" > $FAILFILE_DIR/fail1
mrp --autoretry=1 --psdir=ar_pipeline_test pipeline.mro pipeline_test
