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
    mario.initialize(sys.argv, "join")

    args = mario.Record(mario.metadata.read("args"))
    outs = mario.Record(mario.metadata.read("outs"))
    chunk_defs = [mario.Record(chunk_def) for chunk_def in mario.metadata.read("chunk_defs")]
    chunk_outs = [mario.Record(chunk_out) for chunk_out in mario.metadata.read("chunk_outs")]

    # Execute stage code.
    mario.run("mario.module.join(args, outs, chunk_defs, chunk_outs)")

    # Write the output as JSON.
    mario.metadata.write("outs", outs.items())

    # Write end of log and completion marker.
    mario.complete()

except Exception as e:
    # If stage code threw an error, package it up as JSON.
    mario.fail(mario.stacktrace())
