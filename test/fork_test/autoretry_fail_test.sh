#!/bin/bash
MROPATH=$PWD
PATH=../../bin:$PATH
touch fail1
mrp --autoretry=1 pipeline.mro pipeline_test
