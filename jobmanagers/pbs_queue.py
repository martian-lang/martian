#!/usr/bin/env python
#
# Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
#

"""Queries qstat about a list of jobs and parses the output, returning the list
of jobs which are queued, running, or on hold."""

import subprocess
import sys
import json

# PBS Pro "job states" to be regarded as "alive"
ALIVE = {'Q', 'H', 'W', 'S', 'R', 'E'}


def get_ids():
    """Returns the set of jobids to query from standard input."""
    ids = []
    for jobid in sys.stdin:
        ids.extend(jobid.split())
    return ids


def mkopts(ids):
    """Gets the command line for qstat."""
    if not ids:
        sys.exit(0)
    return ['qstat', '-x', '-F', 'json', '-f'] + ids


def execute(cmd):
    """Executes qstat and captures its output."""
    with subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE) as proc:
        out, err = proc.communicate()
        if proc.returncode:
            raise OSError(err)
        if not isinstance(out, str):
            out = out.decode()
        if len(out) < 500:
            sys.stderr.write(out)
        else:
            sys.stderr.write(out[:496] + '...')
        return out


def parse_output(out):
    """Parses the JSON-format output of qstat and yields the ids of pending
    jobs."""
    data = json.loads(out)
    for jid, info in data.get('Jobs', {}).items():
        if info.get('job_state') in ALIVE:
            yield jid


def main():
    """Reads a set of ids from standard input, queries qstat, and outputs the
    jobids to standard output for jobs which are in the pending state."""
    for jobid in parse_output(execute(mkopts(get_ids()))):
        sys.stdout.write(f'{jobid}\n')
    return 0


if __name__ == '__main__':
    sys.exit(main())
