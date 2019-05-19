#!/usr/bin/env python
#
# Copyright (c) 2017 10X Genomics, Inc. All rights reserved.
#

"""Queries qstat about a list of jobs and parses the output, returning the list
of jobs which are queued, running, or on hold."""

import subprocess
import sys
from xml.etree import cElementTree as ElementTree


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
    return ['qstat', '-s', 'a', '-xml']


def execute(cmd):
    """Executes qstat and captures its output."""
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE)
    out, err = proc.communicate()
    if proc.returncode:
        raise OSError(err)
    sys.stderr.write(out)
    return out


def parse_output(out):
    """Parses the xml-format output of qstat and yields the ids of pending
    jobs."""
    element = ElementTree.fromstring(out)
    for job in list_jobs(element.find('job_info')):
        yield job
    for job in list_jobs(element.find('queue_info')):
        yield job


def list_jobs(jobs):
    """Gets the list of jobs from a job_list."""
    for item in jobs.findall('job_list'):
        if not 'E' in item.find('state').text:
            yield item.find('JB_job_number').text


def main():
    """Reads a set of ids from standard input, queries qstat, and outputs the
    jobids to standard output for jobs which are in the pending state."""
    for jobid in parse_output(execute(mkopts(get_ids()))):
        sys.stdout.write('%s\n' % jobid)
    return 0

if __name__ == '__main__':
    sys.exit(main())
