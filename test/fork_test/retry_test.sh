#!/bin/bash
MROPATH=$PWD
PATH=../../bin:$PATH
touch fail1
mrp pipeline.mro pipeline_test
mrp pipeline.mro pipeline_test
