#!/usr/bin/env python
#
# Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
#
# execute code for an individual stage
#
import sys
import traceback
import mario

try:
    # Initialize Mario with command line args.
    mario.initialize(sys.argv)

    # Load args and retvals from metadata.
    args = mario.Record(mario.metadata.read("args"))

    # Execute split code.
    mario.run("chunk_defs = mario.module.split(args)")

    # Write the output as JSON.
    mario.metadata.write("chunk_defs", chunk_defs)

    # Write end of log and completion marker.
    mario.complete()

except Exception as e:
    # If stage code threw an error, package it up as JSON.
    mario.fail(mario.stacktrace())
