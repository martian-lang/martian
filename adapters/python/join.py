#!/usr/bin/env python
#
# Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
#
# execute code for an individual stage
#
import sys
import traceback
import martian

try:
    # Initialize Martian with command line args.
    martian.initialize(sys.argv)

    # Register handlers for SIGTERM etc.
    martian.setup_signal_handlers()

    args = martian.Record(martian.metadata.read("args"))
    outs = martian.Record(martian.metadata.read("outs"))
    chunk_defs = [martian.Record(chunk_def) for chunk_def in martian.metadata.read("chunk_defs")]
    chunk_outs = [martian.Record(chunk_out) for chunk_out in martian.metadata.read("chunk_outs")]

    # Execute stage code.
    martian.run("martian.module.join(args, outs, chunk_defs, chunk_outs)")

    # Write the output as JSON.
    martian.metadata.write("outs", outs.items())

    # Write end of log and completion marker.
    martian.complete()

except Exception as e:
    # If stage code threw an error, package it up as JSON.
    martian.fail()
