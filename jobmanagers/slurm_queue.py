#!/usr/bin/env python
#
# Copyright (c) 2020 10x Genomics, Inc. All rights reserved.
#

"""Queries squeue about a list of jobs and parses the output, returning the list
of jobs which are queued, running, or on hold."""

import subprocess
import sys


def get_ids():
    """Returns he set of jobids to query from standard input."""
    ids = []
    for jobid in sys.stdin.readlines():
        ids.append(jobid.strip())
    return ids


def mkopts(ids):
    """Gets the command line for qstat."""
    if not ids:
        sys.exit(0)
    return ["squeue", "-o", r"%A %t", "-j", ",".join(ids)]


def execute(cmd):
    """Executes qstat and captures its output."""
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    out, err = proc.communicate()
    if proc.returncode:
        raise OSError(err)
    if not isinstance(out, str):
        out = out.decode()
    if len(out) < 500:
        sys.stderr.write(out)
    else:
        sys.stderr.write(out[:496] + "...")
    return out


def allow_state(state):
    """Returns True if the state code is for a queued or running job."""
    return state in ["CG", "PD", "R", "RD", "RS", "SO"]


def parse_output(out):
    """Parses the output of squeue and yields the ids of pending
    jobs."""
    for line in out.split("\n"):
        if not line:
            continue
        line = line.strip()
        if not line:
            continue
        line = line.split(None, 1)
        if len(line) == 2 and allow_state(line[1]):
            yield line[0]


def main():
    """Reads a set of ids from standard input, queries squeue, and outputs the
    jobids to standard output for jobs which are in the pending state."""
    for jobid in parse_output(execute(mkopts(get_ids()))):
        sys.stdout.write("%s\n" % jobid)
    return 0


if __name__ == "__main__":
    sys.exit(main())
