#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui"
fi
PATH=../../bin:$PATH
echo "1" > fail1
mrp --autoretry=1 pipeline.mro pipeline_test
