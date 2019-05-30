#!/usr/bin/env python
#
# Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
#

# We roll our own six-like py2+3 compatibility to avoid external dependencies.

# This pylint prevents py3 lint from complaining about inheriting from object,
#   and py2 lint from complaining about the "bad" pylint disable option
# pylint: disable=bad-option-value, useless-object-inheritance

"""Martian stage code wrapper.

This module contains infrastructure to load python stage code, possibly
in a python profiling tool, and execute it with appropriate arguments.
Stage code should use the 'martian' module to interface with the
infrastructure.
"""

from __future__ import absolute_import, division, print_function

import os
import sys
import json
import time
import datetime
import errno
import threading
import pstats
import cProfile
import traceback

try:
    from cStringIO import StringIO
except ImportError:
    # Python 3 moved (c)StringIO to the io module
    from io import StringIO

try:
    import line_profiler
except ImportError:
    # Rather than failing here, just assume that line profiling was disabled.
    pass

import martian


#################################################
# Python 2 and 3 compatibility                  #
#################################################


try:
    # py2
    # this pylint disable is because it wants one of these to be UPPERCASE_SNAKE
    #   .. and the other PascalCase, which defeats the purpose of this alias
    # pylint: disable=invalid-name
    _text_type = unicode
    _string_type = basestring
    _PYTHON2, _PYTHON3 = True, False
except NameError:
    # py3
    # pylint: disable=invalid-name
    _text_type = str
    _string_type = str
    _PYTHON2, _PYTHON3 = False, True


#################################################
# Job running infrastructure.                   #
#################################################


class _MemoryProfile(object):
    """Provides a cProfile-like interface for memory profiling."""

    def __init__(self):
        """Initialies the profiler."""
        self.frames = {}
        self.stack = []

    def runcall(self, func, *args, **kwargs):
        """Run a single method under the profiler."""
        sys.setprofile(self._dispatcher)
        try:
            func(*args, **kwargs)
        finally:
            sys.setprofile(None)

    @staticmethod
    def _key(frame, event, arg):
        """Get a key tuple for a frame."""
        fcode = frame.f_code
        caller_fcode = frame.f_back.f_code
        if event == 'c_call':
            filename = arg.__module__
            name = arg.__name__
            ctype = True
        else:
            filename = fcode.co_filename
            name = fcode.co_name
            ctype = False
        return (filename, fcode.co_firstlineno, name, caller_fcode.co_filename,
                caller_fcode.co_firstlineno, caller_fcode.co_name, ctype)

    def _dispatcher(self, frame, event, arg):
        """Callback to collect profile information."""
        if event in ('call', 'c_call'):
            key = self._key(frame, event, arg)
            self.stack.append((key, martian.get_mem_kb()))
        elif event in ('return', 'c_return'):
            key, init_mem_kb = self.stack.pop()
            call_mem_kb = martian.get_mem_kb() - init_mem_kb
            mframe = self.frames.get(key, None)
            if mframe is None:
                self.frames[key] = 1, call_mem_kb, call_mem_kb
            else:
                n_calls, maxrss_kb, total_mem_kb = mframe
                self.frames[key] = n_calls + \
                    1, max(maxrss_kb, call_mem_kb), total_mem_kb + call_mem_kb

    @staticmethod
    def _format_func_name(key):
        """Format the function name from the key info."""
        filename, lineno, name, caller_filename, caller_lineno, caller_name, ctype = key
        func_field_name = 'filename:lineno(function) <--- caller_filename:'\
            'lineno(caller_function)'
        func_caller_str = '%s:%d(%s)' % (
            caller_filename, caller_lineno, caller_name)
        if ctype:
            if filename is None:
                func_name_str = name
            else:
                func_name_str = '%s.%s' % (filename, name)
            return martian.padded_print(
                func_field_name, '{%s} <--- %s' % (func_name_str, func_caller_str))
        return martian.padded_print(func_field_name,
                                    '%s:%d(%s) <--- %s' % (filename,
                                                           lineno,
                                                           name,
                                                           func_caller_str))

    @staticmethod
    def _format_row(key, val):
        """Format a stats row as a string."""
        n_calls, maxrss_kb, total_mem_kb = val

        n_calls_str = martian.padded_print('ncalls', n_calls)
        maxrss_kb_str = martian.padded_print('maxrss(kb)', maxrss_kb)
        total_mem_kb_str = martian.padded_print('totalmem(kb)', total_mem_kb)
        per_call_kb_str = martian.padded_print(
            'percall(kb)', total_mem_kb / n_calls if n_calls > 0 else 0)
        func_str = _MemoryProfile._format_func_name(key)

        return '%s    %s    %s    %s    %s\n' % (
            n_calls_str, maxrss_kb_str, total_mem_kb_str, per_call_kb_str, func_str)

    def format_stats(self):
        """Formats the profile information as a string."""
        sorted_frames = sorted(self.frames.items(),
                               key=lambda frame: frame[1][1], reverse=True)
        output = 'ncalls    maxrss(kb)    totalmem(kb)    percall(kb)    '\
            'filename:lineno(function) <--- '\
            'caller_filename:lineno(caller_function)\n'
        for key, val in sorted_frames:
            output += self._format_row(key, val)
        return output

    def print_stats(self):
        """Prints the profile information to standard out."""
        print(self.format_stats())

    def dump_stats(self, filename):
        """Prints the profilie information to a file with the given name."""
        with open(filename, 'w') as stats_file:
            stats_file.write(self.format_stats())


_METADATA_PREFIX = '_'


class _Metadata(object):
    """Utility methods to read and write martian metadata files used for
    communication with the parent martian instance."""

    def __init__(self, path, files_path, journal_prefix, test=False):
        """Initialize the instance.

        Args:
            path:           The path for metadata communication files.
            files_path:     The path for stage input/output files.
            journal_prefix: The prefix for journal files.
        """
        self.path = path
        self.files_path = files_path
        self.journal_prefix = journal_prefix

        if test:
            self._logfile = sys.stdout
        else:
            self._logfile = os.fdopen(3, 'a')
        self.cache = set()

    def make_path(self, name):
        """Returns a full file path for a named metadata file."""
        return os.path.join(self.path, _METADATA_PREFIX + name)

    @staticmethod
    def make_timestamp(epochsecs):
        """Formats a timestamp according to the martian time format."""
        return datetime.datetime.fromtimestamp(epochsecs).strftime('%Y-%m-%d %H:%M:%S')

    def make_timestamp_now(self):
        """Formats the current time as a string."""
        return self.make_timestamp(time.time())

    def read(self, name):
        """Read the given metadata file as a json file."""
        with open(self.make_path(name), 'r') as source:
            try:
                return json.load(source)
            except ValueError as read_error:
                sys.stderr.write(str(read_error))
                return {}

    def write_raw(self, name, text, force=False):
        """Write the given text to the given metadata file."""
        if isinstance(text, _text_type):
            text = text.encode('utf-8')
        with open(self.make_path(name), 'wb') as dest:
            dest.write(text)
        self.update_journal(name, force)

    def write(self, name, obj=None, force=False):
        """Write the given object to the given metadata file as json."""
        self.write_raw(name,
                       martian.json_dumps_safe(obj or '', indent=4),
                       force)

    def write_raw_atomic(self, name, text, force=False):
        """Write the given text to the given metadata file, by creating a
        temporary file and then moving it, in order to prevent corruption
        of the existing file if the proces of writing is interupted."""
        if isinstance(text, _text_type):
            text = text.encode('utf-8')
        fname = self.make_path(name)
        fname_tmp = fname + '.tmp'
        with open(fname_tmp, 'wb') as dest:
            dest.write(text)
        try:
            os.rename(fname_tmp, fname)
        except OSError as err:
            if err.errno == errno.ENOENT:
                self.log('warn',
                         'Ignoring error moving temp-file %s' % err)
            else:
                raise
        self.update_journal(name, force)

    def write_atomic(self, name, obj, force=False):
        """Write the given object to the given metadata file, by creating a
        temporary file and then moving it, in order to prevent corruption
        of the existing file if the proces of writing is interupted."""
        self.write_raw_atomic(name,
                              martian.json_dumps_safe(obj, indent=4),
                              force)

    def write_time(self, name):
        """Write the current time to th given metadata file."""
        self.write_raw(name, self.make_timestamp_now())

    def _append(self, message, filename):
        """Append to the given metadata file."""
        if isinstance(message, _text_type):
            message = message.encode('utf-8')
        with open(self.make_path(filename), 'a') as dest:
            dest.write(message + '\n')
        self.update_journal(filename)

    @staticmethod
    def _to_string_type(message):
        if _PYTHON3 and isinstance(message, bytes):
            message = message.decode('utf-8', errors='ignore')
        elif not isinstance(message, _string_type):
            # If not a basestring (str or unicode), convert to string here
            message = str(message)
        elif _PYTHON2 and isinstance(message, unicode):  # pylint: disable=undefined-variable
            message = message.encode('utf-8')
        return message

    def log(self, level, message):
        """Write a log line to the log file."""
        self._logfile.write('{} [{}] {}\n'.format(
            self.make_timestamp_now(), level, self._to_string_type(message)))
        self._logfile.flush()

    def alarm(self, message):
        """Append to the alarms file."""
        self._append(message, 'alarm')

    def progress(self, message):
        """Report a progress update."""
        self.write_raw_atomic('progress', message, True)

    @classmethod
    def write_errors(cls, message):
        """Write to the errors file."""
        with os.fdopen(4, 'w') as error_out:
            error_out.write(cls._to_string_type(message))

    @classmethod
    def write_assert(cls, message):
        """Write to the assert file."""
        cls.write_errors('ASSERT:' + cls._to_string_type(message))

    def update_journal(self, name, force=False):
        """Write a journal entry notifying the parent mrp process of changes to
        a given file."""
        if self.journal_prefix and (force or name not in self.cache):
            run_file = self.journal_prefix + name
            tmp_run_file = run_file + '.tmp'
            with open(tmp_run_file, 'w') as tmp_file:
                tmp_file.write(self.make_timestamp_now())
            try:
                os.rename(tmp_run_file, run_file)
            except OSError as err:
                if err.errno == errno.ENOENT:
                    self.log('warn',
                             'Ignoring error moving temp-file %s' % err)
                else:
                    raise
            self.cache.add(name)


class _TestMetadata(_Metadata):
    """A fake metadata object for unit testing."""

    @staticmethod
    def write_errors(message):
        """Write to the errors file."""
        sys.stderr.write(message)


class _CachedJobInfo(object):
    """Stores a subset of jobinfo data which is worth caching."""

    def __init__(self, jobinfo):
        # Cache the profiling and stackvars flags.
        self._profile_mode = jobinfo['profile_mode']
        self._stackvars_flag = (jobinfo['stackvars_flag'] == 'stackvars')

        # Cache invocation and version JSON.
        self._invocation = jobinfo['invocation']
        self._version = jobinfo['version']
        self._threads = jobinfo['threads']
        self._memgb = jobinfo['memGB']

    @property
    def profile_mode(self):
        """The type of in-process profiling to use."""
        return self._profile_mode

    @property
    def stackvars_flag(self):
        """True if all stack variables should be printed on exception."""
        return self._stackvars_flag

    @property
    def invocation(self):
        """The stage invocation object."""
        return self._invocation

    @property
    def version(self):
        """The martian and pipline version information."""
        return self._version

    @property
    def threads(self):
        """The number of threads allocated to this job."""
        return self._threads

    @property
    def mem_gb(self):
        """The amount of memory allocated to this job."""
        return self._memgb


class StageWrapper(object):
    """This class encapsulates the logic for invoking stage code, possibly
    through a wrapper, parsing command line arguments, and so on."""

    def __init__(self, argv, test=False):
        """Initialize the object from command line arguments.

        If test is true, then many bits of functionality are disabled
        and interaction with the filesystem is limited.
        """

        # Take options from command line.
        stagecode_path, self._run_type, metadata_path, files_path, run_file = argv[1:]

        # Create metadata object with metadata directory.
        if test:
            self.metadata = _TestMetadata(
                metadata_path, files_path, None, True)
        else:
            if self._run_type == 'main':
                journal_prefix = run_file + '.'
            else:
                journal_prefix = '%s.%s_' % (run_file, self._run_type)
            self.metadata = _Metadata(
                metadata_path, files_path, journal_prefix)

        # Write jobinfo.
        self.jobinfo = _CachedJobInfo(self.write_jobinfo())

        # Initialize functions to be line-profiled.
        self.funcs = []

        self._result = None

        # Allow shells and stage code to import martian easily
        sys.path.append(os.path.dirname(__file__))

        if not test:
            # Load the stage code as a module.
            sys.path[0] = os.path.dirname(stagecode_path)
            self._module = __import__(os.path.basename(stagecode_path))

    @staticmethod
    def done():
        """Exit the process."""
        # sys.exit does not actually exit the process but only exits the thread.
        # If this thread is not the main thread, use os._exit. This won't call
        # cleanup handlers, flush stdio buffers, etc. But calling done() from
        # another thread means the process exited with an error so this is okay.
        # pylint: disable=protected-access
        if isinstance(threading.current_thread(), threading._MainThread):
            sys.exit(0)
        else:
            # pylint: disable=protected-access
            os._exit(0)

    @staticmethod
    def stacktrace():
        """Get a string containing a stack trace from the most recent frame."""
        etype, evalue, trace_next = sys.exc_info()
        stack = ['Traceback (most recent call last):']
        local = False
        while trace_next:
            frame = trace_next.tb_frame
            filename, lineno, name, line = traceback.extract_tb(trace_next, limit=1)[
                0]
            stack.append("  File '%s', line %d, in %s" %
                         (filename, lineno, name))
            if line:
                stack.append('    %s' % line.strip())
            # Only start printing local variables at stage code
            if filename.endswith('__init__.py') and name in ['main', 'split', 'join']:
                local = True
            if local:
                for key, value in frame.f_locals.items():
                    try:
                        stack.append('        %s = %s' % (key, str(value)))
                    # pylint: disable=bare-except
                    except:
                        pass
            trace_next = trace_next.tb_next
        stack += [line.strip()
                  for line in traceback.format_exception_only(etype, evalue)]
        return '\n'.join(stack)

    def fail(self):
        """Write an errors file with the most recent exception and quit."""
        error_message = traceback.format_exc()
        if self.jobinfo.stackvars_flag:
            self.metadata.write_raw('stackvars', self.stacktrace())
        self.metadata.write_errors(error_message)
        self.done()

    def complete(self):
        """Quit."""
        self.done()

    def _run_profiler(self, cmd, profiler, name):
        """Run cmd under the given profile."""
        profiler.runcall(cmd)
        output_path = self.metadata.make_path(name)
        profiler.dump_stats(output_path)
        self.metadata.update_journal(name)

    def _run(self, cmd):
        """Run the given command under the currently configured profiler."""
        if self.jobinfo.profile_mode == 'mem':
            profiler = _MemoryProfile()
            self._run_profiler(cmd, profiler, 'profile_mem_txt')
        elif self.jobinfo.profile_mode == 'line':
            profiler = None
            try:
                profiler = line_profiler.LineProfiler()
            except NameError:
                martian.throw(
                    'Line-level profiling was requested, but line_profiler was not found.')
            for func in self.funcs:
                profiler.add_function(func)
            self._run_profiler(cmd, profiler, 'profile_line_bin')
            iostr = StringIO()
            profiler.print_stats(stream=iostr)
            self.metadata.write_raw('profile_line_txt', iostr.getvalue())
        elif self.jobinfo.profile_mode == 'cpu':
            profiler = cProfile.Profile()
            self._run_profiler(cmd, profiler, 'profile_cpu_bin')
            iostr = StringIO()
            stats = pstats.Stats(
                profiler, stream=iostr).sort_stats('cumulative')
            stats.print_stats()
            self.metadata.write_raw('profile_cpu_txt', iostr.getvalue())
        else:
            if self.jobinfo.profile_mode and self.jobinfo.profile_mode != 'disable':
                # Give the profiler a little bit of time to attach.
                time.sleep(0.5)
            cmd()

    def _record_result(self, cmd):
        """Runs a command and puts its return value in self._result."""
        self._result = cmd()

    def write_jobinfo(self):
        """Add the 'python' metadata to the existing jobinfo file and return
        the content of the file."""
        jobinfo = self.metadata.read('jobinfo')
        jobinfo['python'] = {
            'binpath': sys.executable,
            'version': sys.version
        }
        self.metadata.write_atomic('jobinfo', jobinfo)
        return jobinfo

    def main(self):
        """Parses command line arguments and runs the stage main."""
        # Load args and retvals from metadata.
        args = martian.Record(self.metadata.read('args'))

        if self._run_type == 'split':
            self._run(lambda: self._record_result(
                lambda: self._module.split(args)))
            self.metadata.write('stage_defs', self._result)
            return

        outs = martian.Record(self.metadata.read('outs'))

        if self._run_type == 'main':
            self._run(lambda: self._module.main(args, outs))
        elif self._run_type == 'join':
            chunk_defs = [martian.Record(chunk_def)
                          for chunk_def in self.metadata.read('chunk_defs')]
            chunk_outs = [martian.Record(chunk_out)
                          for chunk_out in self.metadata.read('chunk_outs')]
            self._run(lambda: self._module.join(
                args, outs, chunk_defs, chunk_outs))
        else:
            martian.throw('Invalid run type %s' % self._run_type)

        # Write the output as JSON.
        self.metadata.write('outs', outs.items())


#################################################
# Executable shell.                             #
#################################################

def _initialize(argv):
    """Initialize global values from the given command line."""
    # pylint: disable=protected-access
    martian._INSTANCE = StageWrapper(argv)
    # pylint: disable=protected-access
    return martian._INSTANCE


def _main(argv):
    """Parse command line and run."""
    stage = None
    try:
        # Initialize Martian with command line args.
        stage = _initialize(argv)

        # Run the stage code.
        stage.main()

        # Exit, making sure to clean up all threads.
        stage.complete()

    # pylint: disable=broad-except
    except Exception:
        # If stage code threw an error, package it up as JSON.
        if stage:
            stage.fail()
        else:
            raise


if __name__ == '__main__':
    _main(sys.argv)
