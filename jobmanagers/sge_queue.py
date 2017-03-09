#!/usr/bin/env python
#
# Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
#

"""Queries qstat about a list of jobs and parses the output, returning the list
of jobs in the queued state."""

import subprocess
import sys
from xml.etree import ElementTree


def get_ids():
    """Returns he set of jobids to query from standard input."""
    ids = []
    for jobid in sys.stdin.readlines():
        ids.append(jobid.strip())


def mkopts(ids):
    """Gets the command line for qstat."""
    if len(ids) == 0:
        sys.exit(0)
    return ['qstat', '-s', 'p', '-xml']


def execute(cmd):
    """Executes qstat and captures its output."""
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE)
    out, err = proc.communicate()
    if proc.returncode:
        raise OSError(err)
    return out


def parse_output(out):
    """Parses the xml-format output of qstat and yields the ids of pending
    jobs."""
    element = ElementTree.fromstring(out)
    jobs = element.find('job_info').findall('job_list')
    for item in jobs:
        if item.get('state') == 'pending':
            yield item.find('JB_job_number').text


def main():
    """Reads a set of ids from standard input, queries qstat, and outputs the
    jobids to standard output for jobs which are in the pending state."""
    for jobid in parse_output(execute(mkopts(get_ids))):
        sys.stdout.write('%s\n' % jobid)
    return 0

if __name__ == '__main__':
    sys.exit(main())
