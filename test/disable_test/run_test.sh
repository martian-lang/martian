#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui --localmem=1 --localcores=1"
fi
PATH=../../bin:$PATH
mrp pipeline.mro pipeline_test
