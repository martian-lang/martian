#!/bin/bash
MROPATH=$PWD
PATH=$PWD/../../bin:$PATH
mrp --overrides=overrides.json pipeline.mro pipeline_test
