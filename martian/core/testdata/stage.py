#!/usr/bin/env python

""" Trivial stage code that parrots inputs to outputs.

The catch is that it does this without the help of the adapter, so that
it can run as part of a go unit test without mrjob.
"""

import errno
import json
import os.path
import sys


def journal(metadata_path, journal_prefix, name, content):
    if hasattr(content, "encode"):
        content = content.encode("utf-8")
    with open(os.path.join(metadata_path, "_" + name), "wb") as log:
        log.write(content)
    try:
        with open(journal_prefix + name, "wb") as tmp_file:
            tmp_file.write(content)
    except (IOError, OSError) as err:
        if err.errno == errno.ENOENT:
            raise


def main(argv):
    """Runs the stage."""
    metadata_path = argv[2]
    run_file = argv[4]
    journal_prefix = run_file + "."
    try:
        journal(metadata_path, journal_prefix, "log", "start\n")
        with open(os.path.join(metadata_path, "_args"), "rb") as args_file:
            args = json.load(args_file)
        outs = {"result": args["what"]}
        with open(os.path.join(metadata_path, "_outs"), "w") as outs_file:
            json.dump(outs, outs_file)
        journal(metadata_path, journal_prefix, "log", "end\n")
        journal(metadata_path, journal_prefix, "complete", "complete\n")
    except Exception as ex:
        journal(metadata_path, journal_prefix, "errors", str(ex))


if __name__ == "__main__":
    main(sys.argv)
