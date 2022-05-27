#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui --jobmode=fake_remote --maxjobs=1"
fi
PATH=../../bin:$PATH
mkdir -p ar_remote_pass
export FAILFILE_DIR=$PWD/ar_remote_pass
echo "1" > $FAILFILE_DIR/fail1
echo "3" > $FAILFILE_DIR/fail3
echo "4" > $FAILFILE_DIR/fail4
mrp --autoretry=3 --psdir=ar_remote_pipeline_test pipeline.mro pipeline_test
