#!/bin/bash
MROPATH=$PWD
if [ -z "$MROFLAGS" ]; then
    export MROFLAGS="--disable-ui --vdrmode=rolling"
fi
PATH=../../bin:$PATH
mrp call.mro pipeline_test
