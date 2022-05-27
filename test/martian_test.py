#!/usr/bin/env python3
#
# Copyright (c) 2016 10x Genomics, Inc. All rights reserved.
#

# This pylint disagrees with black's formatting.
# pylint: disable=bad-option-value, bad-continuation


"""Script to run a test script and compare the output to an expected result.

Includes logic to ignore differences we expect from pipeline outputs,
such as timestamps, versions, and perf information.
"""

from __future__ import absolute_import, division, print_function

import json
import argparse
import os
import re
import shutil
import subprocess
import sys

from fnmatch import fnmatchcase

# We roll our own six-like py2+3 compatibility to avoid external dependencies.
try:
    # py2
    from itertools import izip_longest as zip_longest
except ImportError:
    # py3
    from itertools import zip_longest


try:
    # py2
    # this pylint disable is because it wants one of these to be UPPERCASE_SNAKE
    #   .. and the other PascalCase, which defeats the purpose of this alias
    # pylint: disable=invalid-name
    text_type = unicode
except NameError:
    # py3
    # pylint: disable=invalid-name
    text_type = str


def get_expectation_dir(config_filename, config):
    """Gets the absolute path of the 'expected output' directory from the
    config file."""
    if "expected_dir" in config:
        return os.path.abspath(
            os.path.join(
                os.path.dirname(config_filename), config["expected_dir"]
            )
        )
    return os.path.abspath(
        os.path.join(os.path.dirname(config_filename), "expected")
    )


def get_output_dir(config):
    """Computes the absolute path of the test pipeline work directory from the
    config."""
    if "output_dir" in config:
        return os.path.abspath(
            os.path.join(config["work_dir"], config["output_dir"])
        )
    return None


def expand_glob(root, pattern):
    """Finds all of the files and directories matching pattern, relative to
    root.

    For example, if root is /mnt/awesome and pattern is 'foo/*/_outs' it might
    return foo/bar/_outs, foo/baz/_outs, foo/bar/baz/_outs and so on.

    This would be unnessessary in python3 because glob understands the
    ** recursive wildcard syntax, but in python 2 it is needed.

    Args:
        root (str): The path in which to search.
        pattern (str): The glob pattern to search for relative to the root.
    """
    for cur, dirnames, filenames in os.walk(root):
        for name in filenames:
            if fnmatchcase(
                os.path.relpath(os.path.join(cur, name), root), pattern
            ):
                yield os.path.relpath(os.path.join(cur, name), root)
        for name in dirnames:
            if fnmatchcase(
                os.path.relpath(os.path.join(cur, name), root), pattern
            ):
                yield os.path.relpath(os.path.join(cur, name), root)


_UNIQUIFIER_REGEX = re.compile(
    "(.*%s[a-z]+[0-9]*)-u[0-9a-z]+(%s.+)" % (os.path.sep, os.path.sep)
)


def deuniquify(value):
    """Remove absolute paths and timestamps."""

    def uniquerepl(match):
        """Just take the matched group."""
        return "%s%s" % (match.group(1), match.group(2))

    return _UNIQUIFIER_REGEX.sub(uniquerepl, value)


_FILES_SEP = "%sfiles%s" % (os.path.sep, os.path.sep)


def _parent_files(value):
    """Convert file names like .../chnk0/files/foo to .../files/foo."""
    if not _FILES_SEP in value:
        return value
    if not os.path.basename(value) or not os.path.dirname(value):
        return value
    return os.path.join(
        os.path.dirname(os.path.dirname(os.path.dirname(value))),
        "files",
        os.path.basename(value),
    )


def report_missing(item_type, output, expect, filename, reverse=False):
    """Print a message indicating a missing file."""
    if reverse:
        sys.stderr.write(
            "Extra %s %s\n" % (item_type, os.path.join(expect, filename))
        )
    else:
        sys.stderr.write(
            "Missing %s %s\n" % (item_type, os.path.join(output, filename))
        )


def check_exists_file(output, expect, filename, reverse=False):
    """Checks if a file exists in the output directory."""
    if reverse:
        filename = deuniquify(filename)
    if not os.path.isfile(os.path.join(output, filename)):
        if reverse:
            parent_filename = _parent_files(filename)
            if os.path.isfile(os.path.join(expect, parent_filename)):
                if not os.path.isfile(os.path.join(output, parent_filename)):
                    report_missing(
                        "file", output, expect, parent_filename, reverse
                    )
                    return False
            else:
                report_missing("file", output, expect, filename, reverse)
                return False
        else:
            report_missing("file", output, expect, filename, reverse)
            return False
    return True


def check_exists(output, expect, filename, reverse=False):
    """Checks that a given file, directory, or link in the expected directory
    also exists in the output directory."""
    if os.path.basename(filename).startswith(".nfs"):
        return True  # These are temporary files created by nfs.
    if os.path.isdir(os.path.join(expect, filename)):
        return True  # git does not preserve empty directories
    if os.path.isfile(os.path.join(expect, filename)):
        return check_exists_file(output, expect, filename, reverse)
    if os.path.islink(os.path.join(expect, filename)):
        if not os.path.islink(os.path.join(output, filename)):
            report_missing("link", output, expect, filename, reverse)
            return False
    else:
        raise Exception("No file %s" % os.path.join(expect, filename))
    return True


def compare_dicts(actual, expected, keys):
    """Compares selected keys from two dictionaries."""
    if not actual:
        return not expected
    if not expected:
        return not actual
    for key in keys:
        if key in actual and key in expected:
            if not compare_objects(actual[key], expected[key]):
                sys.stderr.write(
                    "{}: {} != {}\n".format(key, actual[key], expected[key])
                )
                return False
        elif key in expected:
            sys.stderr.write("Missing key: %s\n" % key)
            return False
        elif key in actual:
            sys.stderr.write("Extra key: %s\n" % key)
            return False
        else:
            # if the key is absent from both, we don't care!
            pass
    return True


def _mysorted(lst):
    """Just like sorted, but can also sorts dicts.

    Also handles comparing NoneType with other types (undefined but
    consistent order).
    """
    if not lst:
        return lst
    try:
        return sorted(lst)
    except TypeError:
        if lst is None:
            return lst

        def _mk_key(element):
            if isinstance(element, list):
                return ("list", [_mk_key(i) for i in element])
            if isinstance(element, tuple):
                return ("tuple", (_mk_key(i) for i in element))
            if isinstance(element, dict):
                return ("dict", [(k, _mk_key(v)) for k, v in element.items()])
            return (repr(type(element)), repr(element))

        return sorted(lst, key=_mk_key)


def compare_objects(actual, expected):
    """Compares two objects."""
    if not isinstance(actual, type(expected)):
        sys.stderr.write(
            "Different types: %s != %s\n" % (type(actual), type(expected))
        )
        return False
    if (
        isinstance(actual, text_type)
        and isinstance(expected, text_type)
        or isinstance(actual, str)
        and isinstance(expected, str)
    ):
        if clean_value(actual) != clean_value(expected):
            sys.stderr.write(
                "Different strings: %s != %s\n"
                % (clean_value(actual), clean_value(expected))
            )
            return False
    elif isinstance(actual, dict):
        return compare_dicts(actual, expected, actual.keys()) and compare_dicts(
            actual, expected, expected.keys()
        )
    elif isinstance(actual, list) and isinstance(expected, list):
        for actual_item, expected_item in zip_longest(
            _mysorted(actual), _mysorted(expected)
        ):
            if not compare_objects(actual_item, expected_item):
                return False
    elif actual != expected:
        sys.stderr.write("%s != %s\n" % (actual, expected))
        return False
    return True


def load_json(output, expect, filename):
    """Get two objects to compare from json."""
    try:
        with open(os.path.join(output, filename), "rb") as act:
            actual = json.load(act)
        with open(os.path.join(expect, filename), "rb") as exp:
            expected = json.load(exp)
    except IOError as err:
        sys.stderr.write("Error reading %s: %s\n" % (filename, err))
        return None, None, False
    except ValueError as err:
        sys.stderr.write("%s was not valid json: %s\n" % (filename, err))
        return None, None, False
    except TypeError as err:
        sys.stderr.write(
            "%s contained invalid json types: %s\n" % (filename, err)
        )
        return None, None, False
    return actual, expected, True


def compare_json(output, expect, filename, keys=None):
    """Compare two files which are expected to be json-serialized objects."""
    actual, expected, loaded = load_json(output, expect, filename)
    if not loaded:
        return False
    if keys:
        return compare_dicts(actual, expected, keys)
    return compare_objects(actual, expected)


def compare_jobinfo(output, expect, filename):
    """Compare two _jobinfo json files.  Only compares keys which are expected
    to remain the same across all runs.

    In particular we do not compare rusage, various version, host, cwd,
    and time-related keys, or invocation (which can contain absolute
    paths).
    """
    return compare_json(
        output, expect, filename, ["name", "threads", "memGB", "type",]
    )


def compare_finalstate(output, expect, filename):
    """Compare two _finalstate json files.  Only compares keys within each
    element which are expected to remain the same across runs.

    In particular, we do not compare path, metadata, sweepbindings, or
    forks, all of which may contain absolute paths.
    """
    actual, expected, loaded = load_json(output, expect, filename)
    if not loaded:
        return False
    for actual_info, expected_info in zip_longest(actual, expected):
        if not compare_dicts(
            actual_info,
            expected_info,
            [
                "name",
                "fqname",
                "type",
                "state",
                "edges",
                "stagecodeLang",
                "error",
            ],
        ):
            return False
    return True


def compare_vdrkill(output, expect, filename):
    """Compare two _vdrkill json files.

    We do not compare events or the timestamp
    """
    actual, expected, loaded = load_json(output, expect, filename)
    if not loaded:
        return False
    # size can be different b/c of py2/3 json whitespace differences...
    if not compare_dicts(actual, expected, ["count", "paths", "errors"]):
        return False
    return True


_TIMESTAMP_REGEX = re.compile(
    "[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{1,2}:[0-9]{2}:[0-9]{2}"
)


_QUOTED_PATH_REGEX = re.compile('"/.*/([^/]+)"')
_PATH_REGEX = re.compile("^/.*/([^/]+)$")


def clean_value(value):
    """Remove absolute paths and timestamps."""
    if not value:
        return value

    def pathrepl(match):
        """Just take the matched group."""
        return "%s" % match.group(1)

    unpath = _PATH_REGEX.sub(pathrepl, value)
    if not unpath:
        return unpath
    return _TIMESTAMP_REGEX.sub("__TIMESTAMP__", unpath)


def clean_line(line):
    """Remove absolute paths and timestamps."""

    def pathrepl(match):
        """Just take the matched group."""
        return '"%s"' % match.group(1)

    unpath = _QUOTED_PATH_REGEX.sub(pathrepl, line)
    if not unpath:
        return unpath
    return _TIMESTAMP_REGEX.sub("__TIMESTAMP__", unpath)


def compare_lines(output, expect, filename):
    """Compare two files, replacing everything that might be an absolute path
    with the base path, and timestamps with __TIMESTAMP__."""
    with open(os.path.join(output, filename), "rb") as act:
        with open(os.path.join(expect, filename), "rb") as exp:
            for actual, expected in zip_longest(act, exp):
                if actual and expected:
                    actual = clean_line(
                        actual.decode("utf-8", errors="replace")
                    )
                    expected = clean_line(
                        expected.decode("utf-8", errors="replace")
                    )
                    if actual != expected:
                        sys.stderr.write(
                            "Expected:\n%s\nActual:\n%s\n" % (expected, actual)
                        )
                        return False
    return True


_PPROF_LINE_REGEX = re.compile(r"^# (\S+)")


def pprof_keys(lines):
    """Get the sequence of pprof keys in a file."""
    for line in lines:
        line = line.decode("utf-8", errors="ignore")
        match = _PPROF_LINE_REGEX.match(line)
        if match and match.group(1):
            yield match.group(1)


def compare_pprof(output, expect, filename):
    """Compare two pprof files, only paying attention to keys."""
    with open(os.path.join(output, filename), "rb") as act:
        with open(os.path.join(expect, filename), "rb") as exp:
            for actual, expected in zip_longest(
                pprof_keys(act), pprof_keys(exp)
            ):
                if expected and actual != expected:
                    sys.stderr.write(
                        "Expected:\n%s\nActual:\n%s\n"
                        % (clean_line(expected), clean_line(actual))
                    )
                    return False
    return True


_TRACEBACK_REGEX = re.compile(r", line \d+, in")


def clean_errors(line):
    """As clean_line, but also blur out traceback line numbers."""
    return _TRACEBACK_REGEX.sub(", line LINENO, in", clean_line(line))


def compare_errors(output, expect, filename):
    """As compare_lines, but we also ignore traceback line-numbers."""
    with open(os.path.join(output, filename), "rb") as act:
        with open(os.path.join(expect, filename), "rb") as exp:
            for actual, expected in zip_longest(act, exp):
                if actual and expected:
                    actual = actual.decode("utf-8", errors="ignore")
                    expected = expected.decode("utf-8", errors="ignore")
                    if clean_errors(actual) != clean_errors(expected):
                        sys.stderr.write(
                            "Expected:\n%s\nActual:\n%s\n"
                            % (clean_errors(expected), clean_errors(actual))
                        )
                        return False
    return True


def _compare_true(*_):
    """Sometimes we don't want to compare the actual contents of files."""
    return True


_SPECIAL_FILES = {
    "_perf": _compare_true,
    "_uuid": _compare_true,
    "_versions": _compare_true,
    "_log": _compare_true,
    "_assert": compare_errors,
    "_errors": compare_errors,
    "_jobinfo": compare_jobinfo,
    "_finalstate": compare_finalstate,
    "_vdrkill": compare_vdrkill,
    "_outs": compare_json,
    "_args": compare_json,
    "_stage_defs": compare_json,
    "_vdrkill.partial": compare_json,
}


_SPECIAL_SUFFIXES = {
    ".json": compare_json,
    ".pprof": compare_pprof,
}


def compare_file_content(output, expect, filename):
    """Compare two files.

    Return True if they match.
    """
    base = os.path.basename(filename)
    if base in _SPECIAL_FILES:
        return _SPECIAL_FILES[base](output, expect, filename)
    _, base = os.path.splitext(base)
    if base in _SPECIAL_SUFFIXES:
        return _SPECIAL_SUFFIXES[base](output, expect, filename)
    return compare_lines(output, expect, filename)


def compare_content(output, expect, filename):
    """Check that two paths contain the same content if they are files.

    Does not check anything about non-file objects.
    """
    if not os.path.isfile(os.path.join(expect, filename)):
        if os.path.isfile(os.path.join(output, filename)):
            sys.stderr.write(
                "File should not exist: %s\n" % os.path.join(output, filename)
            )
            return False
    elif not os.path.isfile(os.path.join(output, filename)):
        sys.stderr.write("Missing file %s\n" % os.path.join(output, filename))
        return False
    else:
        if not compare_file_content(output, expect, filename):
            sys.stderr.write(
                "File content mismatch: %s\n"
                "Expected content: %s\n"
                "Actual content: %s\n"
                % (
                    filename,
                    os.path.join(expect, filename),
                    os.path.join(output, filename),
                )
            )
            return False
    return True


def check_result(output_dir, expectation_dir, config):
    """Given an output directory and an expected output directory, and a config
    file containing the tests to apply, checks that the configured success
    criteria all pass."""
    result_ok = True
    if "contains_files" in config:
        for pat in config["contains_files"]:
            for fname in expand_glob(expectation_dir, pat):
                result_ok = (
                    check_exists(output_dir, expectation_dir, fname)
                    and result_ok
                )
    if "contains_only_files" in config:
        for pat in config["contains_only_files"]:
            for fname in expand_glob(expectation_dir, pat):
                result_ok = (
                    check_exists(output_dir, expectation_dir, fname)
                    and result_ok
                )
            for fname in expand_glob(output_dir, pat):
                result_ok = (
                    check_exists(
                        expectation_dir, output_dir, fname, reverse=True
                    )
                    and result_ok
                )
    if "contents_match" in config:
        for pat in config["contents_match"]:
            for fname in expand_glob(expectation_dir, pat):
                result_ok = (
                    compare_content(output_dir, expectation_dir, fname)
                    and result_ok
                )
    return result_ok


def main(argv):
    """Execute the test case."""
    parser = argparse.ArgumentParser()
    parser.add_argument("config")
    config_filename = parser.parse_args(argv[1:]).config
    with open(config_filename, "rb") as configfile:
        config = json.load(configfile)
    if not "command" in config or not config["command"]:
        sys.stderr.write("No command specified in %s\n" % config_filename)
    if not "work_dir" in config:
        config["work_dir"] = os.path.dirname(config_filename)
    config["work_dir"] = os.path.abspath(config["work_dir"])
    config["command"][0] = os.path.abspath(
        os.path.join(os.path.dirname(config_filename), config["command"][0])
    )
    output_dir = get_output_dir(config)
    if output_dir and os.path.isdir(output_dir):
        shutil.rmtree(output_dir)
    sys.stderr.write(
        "Running %s in %s.\n"
        % (" ".join(config["command"]), config["work_dir"])
    )
    return_code = subprocess.call(config["command"], cwd=config["work_dir"])
    if "expected_return" in config:
        if return_code != config["expected_return"]:
            sys.stderr.write("Command returned %d\n" % return_code)
            return 2
    elif return_code != 0:
        sys.stderr.write("Command returned %d\n" % return_code)
        return 2
    expectation_dir = get_expectation_dir(config_filename, config)
    if output_dir and expectation_dir:
        correct = check_result(output_dir, expectation_dir, config)
        if correct:
            sys.stderr.write("Output correct.\n")
            return 0
        sys.stderr.write("Output incorrect!\n")
        return 3
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
