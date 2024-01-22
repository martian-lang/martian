#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui"
fi
PATH=../../bin:$PATH
mkdir -p ar_pass
mrp --autoretry=3 --vdrmode=strict pipeline.mro pipeline_test
