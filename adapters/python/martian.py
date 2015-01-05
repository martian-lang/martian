#!/usr/bin/env python
#
# Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
#
# shared martian object
#
import os
import sys
import json
import time
import datetime
import socket
import subprocess
import resource
import pstats
import StringIO
import cProfile
import traceback

class StageException(Exception):
    pass

class Record(object):
    def __init__(self, dict):
        self.slots = dict.keys()
        for field_name in self.slots:
            setattr(self, field_name, dict[field_name])

    def items(self):
        return dict((field_name, getattr(self, field_name)) for field_name in self.slots)

    def __str__(self):
        return str(self.items())

    def __iter__(self):
        for field_name in self.slots:
            yield getattr(self, field_name)

    def __getitem__(self, index):
        return getattr(self, self.slots[index])

    # Hack for pysam, which can't handle unicode.
    def coerce_strings(self):
        for field_name in self.slots:
            value = getattr(self, field_name)
            if isinstance(value, basestring):
                setattr(self, field_name, str(value))

METADATA_PREFIX = "_"

class Metadata:
    def __init__(self, path, files_path, run_file, run_type):
        self.path = path
        self.files_path = files_path
        self.run_file = run_file
        self.run_type = run_type
        self.cache = {}

    def make_path(self, name):
        return os.path.join(self.path, METADATA_PREFIX + name)

    def make_timestamp(self, epochsecs):
        return datetime.datetime.fromtimestamp(epochsecs).strftime('%Y-%m-%d %H:%M:%S')

    def make_timestamp_now(self):
        return self.make_timestamp(time.time())

    def read(self, name):
        f = open(self.make_path(name))
        o = {}
        try:
            o = json.load(f)
        except ValueError as e:
            sys.stderr.write(str(e))
        f.close()
        return o

    def write_raw(self, name, text):
        with open(self.make_path(name), "w") as f:
            f.write(text)
        self.update_journal(name)

    def write(self, name, object=None):
        self.write_raw(name, json.dumps(object or "", indent=4))

    def write_time(self, name):
        self.write_raw(name, self.make_timestamp_now())

    def log(self, level, message):
        with open(self.make_path("log"), "a") as f:
            f.write("%s [%s] %s\n" % (self.make_timestamp_now(), level, message))
        self.update_journal("log")

    def update_journal(self, name):
        if self.run_type != "main":
            name = "%s_%s" % (self.run_type, name)
        if name not in self.cache:
            run_file = "%s.%s" % (self.run_file, name)
            tmp_run_file = "%s.tmp" % run_file
            with open(tmp_run_file, "w") as f:
                f.write(self.make_timestamp_now())
            os.rename(tmp_run_file, run_file)
            self.cache[name] = True

class TestMetadata(Metadata):
    def log(self, level, message):
        print "%s [%s] %s\n" % (self.make_timestamp_now(), level, message)


def test_initialize(path):
    global metadata
    metadata = TestMetadata(path, path)

def initialize(argv):
    global metadata, module, profile_flag, locals_flag, starttime

    # Take options from command line.
    [ shell_cmd, stagecode_path, metadata_path, files_path, run_file, profile_flag, locals_flag ] = argv

    # Create metadata object with metadata directory.
    run_type = os.path.basename(shell_cmd)[:-3]
    metadata = Metadata(metadata_path, files_path, run_file, run_type)

    # Write jobinfo
    write_jobinfo()

    log_time("__start__")
    starttime = time.time()

    # Cache the profiling and locals flags.
    profile_flag = (profile_flag == "profile")
    locals_flag = (locals_flag == "locals")

    # allow shells and stage code to import martian easily
    sys.path.append(os.path.dirname(__file__))

    # Load the stage code as a module.
    sys.path.append(os.path.dirname(stagecode_path))
    module = __import__(os.path.basename(stagecode_path))

def done():
    log_time("__end__")

    # Common to fail() and complete()
    endtime = time.time()
    jobinfo = metadata.read("jobinfo")
    jobinfo["wallclock"] = {
        "start": metadata.make_timestamp(starttime),
        "end": metadata.make_timestamp(endtime),
        "duration_seconds": endtime - starttime
    }

    def rusage_to_dict(ru):
        # Incantation to convert struct_rusage object into a dict.
        return dict((key, ru.__getattribute__(key)) for key in [ attr for attr in dir(ru) if attr.startswith('ru') ])

    jobinfo["rusage"] = {
        "self": rusage_to_dict(resource.getrusage(resource.RUSAGE_SELF)),
        "children": rusage_to_dict(resource.getrusage(resource.RUSAGE_CHILDREN))
    }
    metadata.write("jobinfo", jobinfo)

def stacktrace():
    etype, evalue, tb = sys.exc_info()
    stacktrace = ["Traceback (most recent call last):"]
    local = False
    while tb:
        frame = tb.tb_frame
        filename, lineno, name, line = traceback.extract_tb(tb, limit=1)[0]
        stacktrace.append("  File '%s', line %d, in %s" % (filename ,lineno, name))
        if line:
            stacktrace.append("    %s" % line.strip())
        # Only start printing local variables at stage code
        if filename.endswith("__init__.py") and name in ["main", "split", "join"] and locals_flag:
            local = True
        if local:
            for key, value in frame.f_locals.items():
                try:
                    stacktrace.append("        %s = %s" % (key, str(value)))
                except:
                    pass
        tb = tb.tb_next
    stacktrace += [line.strip() for line in traceback.format_exception_only(etype, evalue)]
    if local:
        stacktrace.append("\n")
        stacktrace.append(traceback.format_exc())
    return "\n".join(stacktrace)

def fail(stacktrace):
    metadata.write_raw("errors", stacktrace)
    done()

def complete():
    metadata.write_time("complete")
    done()

def run(cmd):
    if profile_flag:
        profile = cProfile.Profile()
        profile.enable()
        profile.run(cmd)
        profile.disable()
        str = StringIO.StringIO()
        ps = pstats.Stats(profile, stream=str).sort_stats("cumulative")
        ps.print_stats()
        metadata.write_raw("profile", str.getvalue())
        full_profile_path = metadata.make_path("profile_full")
        profile.dump_stats(full_profile_path)
        metadata.update_journal("profile_full")
    else:
        import __main__
        exec(cmd, __main__.__dict__, __main__.__dict__)

def Popen(args, bufsize=0, executable=None, stdin=None, stdout=None, stderr=None,
    preexec_fn=None, close_fds=False, shell=False, cwd=None, env=None,
    universal_newlines=False, startupinfo=None, creationflags=0):
    metadata.log("exec", " ".join(args))
    return subprocess.Popen(args, bufsize=bufsize, executable=executable, stdin=stdin,
        stdout=stdout, stderr=stderr, preexec_fn=preexec_fn, close_fds=close_fds,
        shell=shell, cwd=cwd, env=env, universal_newlines=universal_newlines,
        startupinfo=startupinfo, creationflags=creationflags)

def check_call(args, stdin=None, stdout=None, stderr=None, shell=False):
    metadata.log("exec", " ".join(args))
    return subprocess.check_call(args, stdin=stdin, stdout=stdout, stderr=stderr, shell=shell)

def make_path(filename):
    return os.path.join(metadata.files_path, filename)

def write_jobinfo():
    jobinfo = metadata.read("jobinfo")
    jobinfo["cwd"] = os.getcwd()
    jobinfo["host"] = socket.gethostname()
    jobinfo["pid"] = os.getpid()
    jobinfo["python"] = {
        "binpath": sys.executable,
        "version": sys.version
    }
    if os.environ.get("SGE_ARCH"):
        jobinfo["sge"] = {
            "root": os.environ.get("SGE_ROOT"),
            "cell": os.environ.get("SGE_CELL"),
            "queue": os.environ.get("QUEUE"),
            "jobid": os.environ.get("JOB_ID"),
            "jobname": os.environ.get("JOB_NAME"),
            "sub_host": os.environ.get("SGE_O_HOST"),
            "sub_user": os.environ.get("SGE_O_LOGNAME"),
            "exec_host": os.environ.get("HOSTNAME"),
            "exec_user": os.environ.get("LOGNAME")
        }
    metadata.write("jobinfo", jobinfo)

def log_info(message):
    metadata.log("info", message)

def log_warn(message):
    metadata.log("warn", message)

def log_time(message):
    metadata.log("time", message)

def log_json(label, object):
    metadata.log("json", json.dumps({"label":label, "object":object}))

def throw(message):
    raise StageException(message)
