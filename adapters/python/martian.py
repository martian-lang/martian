#!/usr/bin/env python
#
# Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
#

"""Martian stage code API and common utility methods.

This module contains an API for python stage code to use to interact
with the higher-level martian logic, plus common utility methods.
"""


import json
import math
import os
import resource
import subprocess
import sys

# Singleton instance object.
if not '_INSTANCE' in globals():
    _INSTANCE = None


class StageException(Exception):
    """Base exception type for stage code."""
    pass


class Record(object):
    """An object with a set of attributes generated from a dictioanry."""

    def __init__(self, f_dict):
        """Initializes the object from a dictionary."""
        self.slots = f_dict.keys()
        for field_name in self.slots:
            setattr(self, field_name, f_dict[field_name])

    def items(self):
        """Returns the a dictionary with the elements which were in the keys in
        dictionary used to initialize the record."""
        return dict((field_name, getattr(self, field_name)) for field_name in self.slots)

    def __str__(self):
        """Formats the object as a string."""
        return str(self.items())

    def __iter__(self):
        """Iterate through the values of the object corresponding to keys in
        the dictioanry used to initialize the object."""
        for field_name in self.slots:
            yield getattr(self, field_name)

    def __getitem__(self, index):
        """Get the value associated with the Nth key in the source
        dictionary."""
        return getattr(self, self.slots[index])

    # Hack for pysam, which can't handle unicode.
    def coerce_strings(self):
        """Convert all basestring values into str values."""
        for field_name in self.slots:
            value = getattr(self, field_name)
            if isinstance(value, basestring):
                setattr(self, field_name, str(value))


def json_sanitize(data):
    """Converts NaN values into None values."""
    if isinstance(data, float):
        # Handle exceptional floats.
        if math.isnan(data) or data == float('+Inf') or data == float('-Inf'):
            return None
        return data
    elif isinstance(data, dict):
        # Recurse on dictionaries.
        new_data = {}
        for k in data.keys():
            new_data[k] = json_sanitize(data[k])
        return new_data
    elif hasattr(data, '__iter__'):
        # Recurse on lists.
        return [json_sanitize(d) for d in data]
    else:
        return data


def json_dumps_safe(data, indent=None):
    """Returns a formatted json string of the data, with NaN values converted
    to None."""
    return json.dumps(json_sanitize(data), indent=indent)


def get_mem_kb():
    """Get the current max rss memory for this process and completed child
    processes."""
    return max(resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
               resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss)


def convert_gb_to_kb(mem_gb):
    """Convert from gb to kb."""
    return mem_gb * 1024 * 1024


def padded_print(field_name, value):
    """Pad a string with leading spaces to be the same length as field_name."""
    offset = len(field_name) - len(str(value))
    if offset > 0:
        return (' ' * offset) + str(value)
    return str(value)


def profile(func):
    """Add a fuction to the set of functions to be covered by the line
    profiler."""
    _INSTANCE.funcs.append(func)
    return func


# On linux, provide a method to set PDEATHSIG on child processes.
if sys.platform.startswith('linux'):
    import ctypes
    import ctypes.util
    from signal import SIGKILL

    _LIBC = ctypes.CDLL(ctypes.util.find_library('c'))

    _PR_SET_PDEATHSIG = ctypes.c_int(1)  # <sys/prctl.h>

    def child_preexec_set_pdeathsig():
        """When used as the preexec_fn argument for subprocess.Popen etc,
        causes the subprocess to recieve SIGKILL if the parent process
        terminates."""
        zero = ctypes.c_ulong(0)
        _LIBC.prctl(_PR_SET_PDEATHSIG, ctypes.c_ulong(SIGKILL),
                    zero, zero, zero)
else:
    child_preexec_set_pdeathsig = None  # pylint: disable=invalid-name


# pylint: disable=invalid-name,too-many-arguments
def Popen(args, bufsize=0, executable=None,
          stdin=None, stdout=None, stderr=None,
          preexec_fn=child_preexec_set_pdeathsig, close_fds=False,
          shell=False, cwd=None, env=None, universal_newlines=False,
          startupinfo=None, creationflags=0):
    """Log opening of a subprocess."""
    _INSTANCE.metadata.log('exec', ' '.join(args))
    return subprocess.Popen(args, bufsize=bufsize, executable=executable,
                            stdin=stdin, stdout=stdout, stderr=stderr,
                            preexec_fn=preexec_fn, close_fds=close_fds,
                            shell=shell, cwd=cwd, env=env,
                            universal_newlines=universal_newlines,
                            startupinfo=startupinfo,
                            creationflags=creationflags)


def check_call(args, stdin=None, stdout=None, stderr=None, shell=False):
    """Log running a given subprocess."""
    _INSTANCE.metadata.log('exec', ' '.join(args))
    return subprocess.check_call(args, shell=shell,
                                 stdin=stdin, stdout=stdout, stderr=stderr,
                                 preexec_fn=child_preexec_set_pdeathsig)


def make_path(filename):
    """Get the file path for a named file."""
    return os.path.join(_INSTANCE.metadata.files_path, filename)


def get_invocation_args():
    """Get the args from the invocation."""
    return _INSTANCE.jobinfo.invocation['args']


def get_invocation_call():
    """Get the call information from the invocation."""
    return _INSTANCE.jobinfo.invocation['call']


def get_martian_version():
    """Get the martian version from the jobinfo."""
    return _INSTANCE.jobinfo.version['martian']


def get_pipelines_version():
    """Get the pipelines version from the jobinfo."""
    return _INSTANCE.jobinfo.version['pipelines']


def get_threads_allocation():
    """Get the number of threads allocated to this job by the runtime."""
    return _INSTANCE.jobinfo.threads


def get_memory_allocation():
    """Get the amount of memory in GB allocated to this job by the runtime."""
    return _INSTANCE.jobinfo.mem_gb


def update_progress(message):
    """Updates the current progress of the stage, which will be displayed to
    the user (in the mrp log) next time mrp reads the file."""
    _INSTANCE.metadata.progress(message)


def log_info(message):
    """Log a message."""
    _INSTANCE.metadata.log('info', message)


def log_warn(message):
    """Log a warning."""
    _INSTANCE.metadata.log('warn', message)


def log_time(message):
    """Log a timestamp for an action."""
    _INSTANCE.metadata.log('time', message)


def log_json(label, obj):
    """Log an object in json format."""
    _INSTANCE.metadata.log('json', json_dumps_safe(
        {'label': label, 'object': obj}))


def throw(message):
    """Raise a stage exception."""
    raise StageException(message)


# pylint: disable=redefined-builtin
def exit(message):
    """Fail the pipeline with an assertion."""
    _INSTANCE.metadata.write_assert(message)
    _INSTANCE.done()


def alarm(message):
    """Add a message to the alarms."""
    _INSTANCE.metadata.alarm(message)


#################################################
# Initialization                                #
#################################################


def test_initialize(path):
    """Initialize with a fake test metadata."""
    import martian_shell as mr_shell

    # pylint: disable=global-statement
    global _INSTANCE
    _INSTANCE = mr_shell.StageWrapper(
        [None, None, 'main', path, path, ''], True)
    return _INSTANCE
