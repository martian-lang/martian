#!/usr/bin/env python
#
# Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
#
# Accelerated stress-tester for Martian runtime.
#
import martian
import time
import random

SLEEPSECS = 0
THREADS = 1
MEMGB = 1.0
CHUNKS = 10
random.seed()

def split(args):
    time.sleep(SLEEPSECS)
    THREADS = random.randint(1,8)
    CHUNKS = random.randint(10,50)
    return [{'__threads': THREADS, '__mem_gb': MEMGB} for i in range(0, CHUNKS)]

def main(args, outs):
    time.sleep(SLEEPSECS)
    pass

def join(args, outs, chunk_defs, chunk_outs):
    time.sleep(SLEEPSECS)
    pass

