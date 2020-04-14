#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui --vdrmode=strict"
fi
PATH=../../bin:$PATH
mrp call.mro pipeline_test
