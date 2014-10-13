#!/usr/bin/env python
#
# Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
#
# Runs alignment code
#
import mario
import time

SLEEPSECS = 1
THREADS = 1
MEMGB = 1.0
CHUNKS = 100

def split(args):
    time.sleep(SLEEPSECS)
    return [{'__threads': THREADS, '__mem_gb': MEMGB} for i in range(0, CHUNKS)]

def main(args, outs):
    time.sleep(SLEEPSECS)
    pass

def join(args, outs, chunk_defs, chunk_outs):
    time.sleep(SLEEPSECS)
    pass

