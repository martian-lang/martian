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
import signal
import socket
import subprocess
import multiprocessing
import resource
import threading
import pstats
import StringIO
import cProfile
import traceback
import line_profiler
import math

def setup_signal_handlers():
    """Registers signal handlers to actually write an error file.
    
    The error string is intended to match what would be written if running in
    local mode.  This prevents the martian runtime from waiting for a
    heartbeat failure if the cluster kills a job for some reason.
    """
    def handler(signum, frame):
        global metadata
        global _done_called
        if _done_called.value != 0:
            signal.signal(signum, signal.SIG_DFL)
            return
        metadata.write_raw("errors", "signal: %d\n\n%s\n" %
                           (signum, ''.join(reversed(
                               traceback.format_stack(frame)))))
        signal.signal(signum, signal.SIG_DFL)  # only catch first signal.
        done()
    # These are the signals which are guaranteed to work on all platforms.
    # They should be enough for the cases we're actually interested in.
    signal.signal(signal.SIGABRT, handler)
    signal.signal(signal.SIGFPE, handler)
    signal.signal(signal.SIGILL, handler)
    signal.signal(signal.SIGINT, handler)
    signal.signal(signal.SIGTERM, handler)

def json_sanitize(data):
    if (type(data) == float):
        # Handle exceptional floats.
        if math.isnan(data):
            return None
        if (data ==  float("+Inf")):
            return None
        if (data == float("-Inf")):
            return None
        return data
    elif type(data) == dict:
        # Recurse on dictionaries.
        new_data = {}
        for k in data.keys():
            new_data[k] = json_sanitize(data[k])
        return new_data
    elif hasattr(data, '__iter__'):
        # Recurse on lists.
        new_data = []
        for d in data:
            new_data.append(json_sanitize(d))
        return new_data
    else:
        return data

def json_dumps_safe(data, indent=None):
    return json.dumps(json_sanitize(data), indent=indent)

class StageException(Exception):
    pass

class MemoryProfile:
    def __init__(self):
        self.frames = {}
        self.stack = []

    def run(self, cmd):
        import __main__
        dict = __main__.__dict__
        self.runctx(cmd, dict, dict)

    def runctx(self, cmd, globals, locals):
        sys.setprofile(self.dispatcher)
        try:
            exec cmd in globals, locals
        finally:
            sys.setprofile(None)

    def dispatcher(self, frame, event, arg):
        if event == "call" or event == "c_call":
            fcode = frame.f_code
            caller_fcode = frame.f_back.f_code
            if event == "c_call":
                filename = arg.__module__
                name = arg.__name__
                ctype = True
            else:
                filename = fcode.co_filename
                name = fcode.co_name
                ctype = False
            key = (filename, fcode.co_firstlineno, name, caller_fcode.co_filename,
                   caller_fcode.co_firstlineno, caller_fcode.co_name, ctype)
            self.stack.append((key, get_mem_kb()))
        elif event == "return" or event == "c_return":
            key, init_mem_kb = self.stack.pop()
            call_mem_kb = get_mem_kb() - init_mem_kb
            mframe = self.frames.get(key, None)
            if mframe is None:
                self.frames[key] = 1, call_mem_kb, call_mem_kb
            else:
                n_calls, maxrss_kb, total_mem_kb = mframe
                self.frames[key] = n_calls + 1, max(maxrss_kb, call_mem_kb), total_mem_kb + call_mem_kb

    def format_stats(self):
        sorted_frames = sorted(self.frames.items(), key=lambda frame: frame[1][1], reverse=True)
        output = "ncalls    maxrss(kb)    totalmem(kb)    percall(kb)    filename:lineno(function) <--- caller_filename:lineno(caller_function)\n"
        for key, val in sorted_frames:
            filename, lineno, name, caller_filename, caller_lineno, caller_name, ctype = key
            n_calls, maxrss_kb, total_mem_kb = val

            n_calls_str = padded_print("ncalls", n_calls)
            maxrss_kb_str = padded_print("maxrss(kb)", maxrss_kb)
            total_mem_kb_str = padded_print("totalmem(kb)", total_mem_kb)
            per_call_kb_str = padded_print("percall(kb)", total_mem_kb / n_calls if n_calls > 0 else 0)

            func_field_name = "filename:lineno(function) <--- caller_filename:lineno(caller_function)"
            func_caller_str = "%s:%d(%s)" % (caller_filename, caller_lineno, caller_name)
            if ctype:
                if filename is None:
                    func_name_str = name
                else:
                    func_name_str = "%s.%s" % (filename, name)
                func_str = padded_print(func_field_name, "{%s} <--- %s" % (func_name_str, func_caller_str))
            else:
                func_str = padded_print(func_field_name, "%s:%d(%s) <--- %s" % (filename, lineno, name, func_caller_str))
            output += "%s    %s    %s    %s    %s\n" % (n_calls_str, maxrss_kb_str, total_mem_kb_str, per_call_kb_str, func_str)
        return output

    def print_stats(self):
        print self.format_stats()

    def dump_stats(self, filename):
        with open(filename, 'w') as f:
            f.write(self.format_stats())

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
        self.write_raw(name, json_dumps_safe(object or "", indent=4))

    def write_time(self, name):
        self.write_raw(name, self.make_timestamp_now())

    def _append(self, message, filename):
        with open(self.make_path(filename), "a") as f:
            f.write(message + "\n")
        self.update_journal(filename)

    def log(self, level, message):
        self._append("%s [%s] %s" % (self.make_timestamp_now(), level, message), "log")

    def alarm(self, message):
        self._append(message, "alarm")

    def _assert(self, message):
        self.write_raw("assert", message)

    def update_journal(self, name, force=False):
        if self.run_type != "main":
            name = "%s_%s" % (self.run_type, name)
        if name not in self.cache or force:
            run_file = "%s.%s" % (self.run_file, name)
            tmp_run_file = "%s.tmp" % run_file
            with open(tmp_run_file, "w") as f:
                f.write(self.make_timestamp_now())
            os.rename(tmp_run_file, run_file)
            self.cache[name] = True

class TestMetadata(Metadata):
    def log(self, level, message):
        print "%s [%s] %s\n" % (self.make_timestamp_now(), level, message)


def get_mem_kb():
    return max(resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,  resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss)

def convert_gb_to_kb(mem_gb):
    return mem_gb * 1024 * 1024

def padded_print(field_name, value):
    offset = len(field_name) - len(str(value))
    if offset > 0:
        return (" " * offset) + str(value)
    return str(value)

def test_initialize(path):
    global metadata
    metadata = TestMetadata(path, path, "", "main")

# Needs to be global so that the GC doesn't eat it.
_heartbeat_process = None
_done_called = multiprocessing.Value('i', 0)

def heartbeat(metadata, done):
    while done.value == 0:
        metadata.update_journal("heartbeat", force=True)
        time.sleep(120)

def monitor(metadata, limit_kb):
    while True:
        maxrss_kb = get_mem_kb()
        if limit_kb < maxrss_kb:
            metadata.write_raw("errors", "Job exceeded memory limit of %d KB. Used %d KB" % (limit_kb, maxrss_kb))
            done()
        time.sleep(120)

def start_heartbeat():
    global _done_called
    t = multiprocessing.Process(target=heartbeat, args=(metadata, _done_called))
    t.daemon = True
    global _heartbeat_process
    _heartbeat_process = t
    t.start()

def start_monitor(limit_kb):
    t = threading.Thread(target=monitor, args=(metadata, limit_kb))
    t.daemon = True
    t.start()

def initialize(argv):
    global metadata, module, profile_mode, stackvars_flag, starttime, invocation, version, funcs

    # Take options from command line.
    shell_cmd, stagecode_path, metadata_path, files_path, run_file = argv

    # Create metadata object with metadata directory.
    run_type = os.path.basename(shell_cmd)[:-3]
    metadata = Metadata(metadata_path, files_path, run_file, run_type)

    # Log the start time (do not call this before metadata is initialized)
    log_time("__start__")
    starttime = time.time()

    # Write jobinfo.
    jobinfo = write_jobinfo(files_path)

    # Update journal for stdout / stderr
    metadata.update_journal("stdout")
    metadata.update_journal("stderr")

    # Start heartbeat thread
    start_heartbeat()

    # Start monitor thread
    monitor_flag = (jobinfo["monitor_flag"] == "monitor")
    limit_kb = convert_gb_to_kb(jobinfo["memGB"])
    if monitor_flag:
        start_monitor(limit_kb)

    # Increase the maximum open file descriptors to the hard limit
    _, hard = resource.getrlimit(resource.RLIMIT_NOFILE)
    try:
        resource.setrlimit(resource.RLIMIT_NOFILE, (hard, hard))
    except Exception as e:
        # Since we are still initializing, do not allow an unhandled exception.
        # If the limit is not high enough, a preflight will catch it.
        metadata.log("adapter", "Adapter could not increase file handle ulimit to %s: %s" % (str(hard), str(e)))
        pass

    # Cache the profiling and stackvars flags.
    profile_mode = jobinfo["profile_mode"]
    stackvars_flag = (jobinfo["stackvars_flag"] == "stackvars")

    # Cache invocation and version JSON.
    invocation = jobinfo["invocation"]
    version = jobinfo["version"]

    # Allow shells and stage code to import martian easily
    sys.path.append(os.path.dirname(__file__))

    # Initialize functions to be line-profiled.
    funcs = []

    # Load the stage code as a module.
    sys.path.append(os.path.dirname(stagecode_path))
    module = __import__(os.path.basename(stagecode_path))

def done():
    log_time("__end__")

    # Stop the heartbeat.
    global _done_called
    global _heartbeat_process
    _done_called.value = 1
    _heartbeat_process = None

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

    # sys.exit does not actually exit the process but only exits the thread.
    # If this thread is not the main thread, use os._exit. This won't call
    # cleanup handlers, flush stdio buffers, etc. But calling done() from
    # another thread means the process exited with an error so this is okay.
    if isinstance(threading.current_thread(), threading._MainThread):
        sys.exit(0)
    else:
        os._exit(0)

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
        if filename.endswith("__init__.py") and name in ["main", "split", "join"]:
            local = True
        if local:
            for key, value in frame.f_locals.items():
                try:
                    stacktrace.append("        %s = %s" % (key, str(value)))
                except:
                    pass
        tb = tb.tb_next
    stacktrace += [line.strip() for line in traceback.format_exception_only(etype, evalue)]
    return "\n".join(stacktrace)

def fail():
    metadata.write_raw("errors", traceback.format_exc())
    if stackvars_flag:
        metadata.write_raw("stackvars", stacktrace())
    done()

def complete():
    metadata.write_time("complete")
    done()

def profile(f):
    funcs.append(f)
    return f

def run_profiler(cmd, profiler, name):
    profiler.run(cmd)
    output_path = metadata.make_path(name)
    profiler.dump_stats(output_path)
    metadata.update_journal(name)

def run(cmd):
    if profile_mode == "mem":
        profiler = MemoryProfile()
        run_profiler(cmd, profiler, "profile_mem_txt")
    elif profile_mode == "line":
        profiler = line_profiler.LineProfiler()
        for f in funcs:
            profiler.add_function(f)
        run_profiler(cmd, profiler, "profile_line_bin")
        str = StringIO.StringIO()
        profiler.print_stats(stream=str)
        metadata.write_raw("profile_line_txt", str.getvalue())
    elif profile_mode == "cpu":
        profiler = cProfile.Profile()
        run_profiler(cmd, profiler, "profile_cpu_bin")
        str = StringIO.StringIO()
        ps = pstats.Stats(profiler, stream=str).sort_stats("cumulative")
        ps.print_stats()
        metadata.write_raw("profile_cpu_txt", str.getvalue())
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

def write_jobinfo(cwd):
    jobinfo = metadata.read("jobinfo")
    jobinfo["cwd"] = cwd
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
    return jobinfo

def get_invocation_args():
    return invocation["args"]

def get_invocation_call():
    return invocation["call"]

def get_martian_version():
    return version["martian"]

def get_pipelines_version():
    return version["pipelines"]

def log_info(message):
    metadata.log("info", message)

def log_warn(message):
    metadata.log("warn", message)

def log_time(message):
    metadata.log("time", message)

def log_json(label, object):
    metadata.log("json", json_dumps_safe({"label":label, "object":object}))

def throw(message):
    raise StageException(message)

def exit(message):
    metadata._assert(message)
    done()

def alarm(message):
    metadata.alarm(message)
