#!/usr/bin/env python
#
# Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
#
# shared mario object
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

class MarioException(Exception):
    pass

class Record(object):
    def __init__(self, dict):
        self.slots = dict.keys()
        for field_name in self.slots:
            setattr(self, field_name, dict[field_name])

    def items(self):
        return dict((field_name, getattr(self, field_name)) for field_name in self.slots)

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
    def __init__(self, path, files_path):
        self.path = path
        self.files_path = files_path

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
        f = open(self.make_path(name), "w")
        f.write(text)
        f.close()

    def write(self, name, object=None):
        self.write_raw(name, json.dumps(object or "", indent=4))

    def write_time(self, name):
        self.write_raw(name, self.make_timestamp_now())

    def log(self, level, message):
        f = open(self.make_path("log"), "a")
        f.write("%s [%s] %s\n" % (self.make_timestamp_now(), level, message))
        f.close()

class TestMetadata(Metadata):
    def log(self, level, message):
        print "%s [%s] %s\n" % (self.make_timestamp_now(), level, message)


def test_initialize(path):
    global metadata
    metadata = TestMetadata(path, path)

def initialize(argv):
    global metadata, module, profile_flag, starttime

    # Take options from command line.
    [ shell_cmd, stagecode_path, lib_path, metadata_path, files_path, profile_flag ] = argv

    # Create metadata object with metadata directory.
    metadata = Metadata(metadata_path, files_path)

    # Write jobinfo
    write_jobinfo()

    log_time("__start__")
    starttime = time.time()

    # Cache the profiling flag.
    profile_flag = (profile_flag == "profile")

    # allow shells and stage code to import mario easily
    sys.path.append(os.path.dirname(__file__))

    # allow stage code to import lib modules
    sys.path.append(lib_path)

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
    raise MarioException(message)
