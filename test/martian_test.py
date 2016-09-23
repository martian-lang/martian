#!/usr/bin/env python
#
# Copyright (c) 2016 10x Genomics, Inc. All rights reserved.
#

"""Script to run a test script and compare the output to an expected result.

Includes logic to ignore differences we expect from pipeline outputs, such as
timestamps, versions, and perf information.
"""

import filecmp
import itertools
import json
import optparse
import os
import re
import shutil
import subprocess
import sys

from fnmatch import fnmatchcase


def ExpectationDir(config_filename, config):
    """Gets the absolute path of the 'expected output' directory from
    the config file."""
    if 'expected_dir' in config:
        return os.path.abspath(os.path.join(os.path.dirname(config_filename),
                                            config['expected_dir']))
    else:
        return os.path.abspath(os.path.join(os.path.dirname(config_filename),
                                            'expected'))


def OutputDir(config):
    """Computes the absolute path of the test pipeline work directory from
    the config."""
    if 'output_dir' in config:
        return os.path.abspath(os.path.join(config['work_dir'],
                                            config['output_dir']))
    else:
        return None


def ExpandGlob(root, pattern):
    """Finds all of the files and directories matching pattern,
    relative to root.

    For example, if root is /mnt/awesome and pattern is 'foo/*/_outs' it might
    return foo/bar/_outs, foo/baz/_outs, foo/bar/baz/_outs and so on.

    This would be unnessessary in python3 because glob understands the
    ** recursive wildcard syntax, but in python 2 it is needed.
    """
    for cur, dirnames, filenames in os.walk(root):
        for fn in filenames:
            if fnmatchcase(os.path.relpath(os.path.join(cur, fn), root), pattern):
                yield os.path.relpath(os.path.join(cur, fn), root)
        for fn in dirnames:
            if fnmatchcase(os.path.relpath(os.path.join(cur, fn), root), pattern):
                yield os.path.relpath(os.path.join(cur, fn), root)


def CheckExists(output, expect, filename):
    """Checks that a given file, directory, or link in the expected directory
    also exists in the output directory."""
    if os.path.isdir(os.path.join(expect, filename)):
        if not os.path.isdir(os.path.join(output, filename)):
            sys.stderr.write('Missing directory %s\n'
                             % os.path.join(output, filename))
            return False
    elif os.path.isfile(os.path.join(expect, filename)):
        if not os.path.isfile(os.path.join(output, filename)):
            sys.stderr.write('Missing file %s\n'
                             % os.path.join(output, filename))
            return False
    elif os.path.islink(os.path.join(expect, filename)):
        if not os.path.islink(os.path.join(output, filename)):
            sys.stderr.write('Missing link %s\n'
                             % os.path.join(output, filename))
            return False
    else:
        raise Exception('No file %s'
                        % os.path.join(expect, filename))
    return True


def CompareDicts(actual, expected, keys):
    """Compares selected keys from two dictionaries."""
    if not actual:
        return not expected
    if not expected:
        return not actual
    for key in keys:
        if key in actual:
            if key in expected:
                if actual[key] != expected[key]:
                    sys.stderr.write('%s: %s != %s\n' %
                                     (key, actual[key], expected[key]))
                    return False
            else:
                sys.stderr.write('Missing key %s\n' % key)
                return False
        elif key in expected:
            sys.stderr.write('Extra key %s\n' % key)
            return False
    return True


def CompareJobinfo(output, expect, filename):
    """Compare two _jobinfo json files.  Only compares keys which are expected
    to remain the same across all runs.

    In particular we do not compare rusage, various version, host, cwd, and
    time-related keys, or invocation (which can contain absolute paths).
    """
    try:
        with open(os.path.join(output, filename)) as act:
            actual = json.load(act)
        with open(os.path.join(expect, filename)) as exp:
            expected = json.load(exp)
    except Exception as err:
        sys.stderr.write('Error reading %s: %s\n' % (filename, err))
        return False
    return CompareDicts(actual, expected, ['name',
                                           'threads',
                                           'memGB',
                                           'type',
                                          ])


def CompareFinalState(output, expect, filename):
    """Compare two _finalstate json files.  Only compares keys within each
    element which are expected to remain the same across runs.

    In particular, we do not compare path, metadata, sweepbindings, or forks,
    all of which may contain absolute paths.
    """
    try:
        with open(os.path.join(output, filename)) as act:
            actual = json.load(act)
        with open(os.path.join(expect, filename)) as exp:
            expected = json.load(exp)
    except Exception as err:
        sys.stderr.write('Error reading %s: %s\n' % (filename, err))
        return False
    for actual_info, expected_info in itertools.izip_longest(actual, expected):
        if not CompareDicts(actual_info, expected_info, ['name',
                                                         'fqname',
                                                         'type',
                                                         'state',
                                                         'edges',
                                                         'stagecodeLang',
                                                         'error',
                                                        ]):
            return False
    return True


_TIMESTAMP_REGEX = re.compile(
    '[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{1,2}:[0-9]{2}:[0-9]{2}')


_PATH_REGEX = re.compile('"/.*/([^/]+)"')


def CompareLines(output, expect, filename):
    """Compare two files, replacing everything that might be an absolute path
    with the base path, and timestamps with __TIMESTAMP__."""
    def clean_line(line):
        """Remove absolute paths and timestamps."""
        def pathrepl(match):
            """Just take the matched group."""
            return '"%s"' % match.group(1)
        return _TIMESTAMP_REGEX.sub('__TIMESTAMP__',
                                    _PATH_REGEX.sub(pathrepl, line))
    with open(os.path.join(output, filename)) as act:
        with open(os.path.join(expect, filename)) as exp:
            for actual, expected in itertools.izip_longest(act, exp):
                if actual and expected:
                  if clean_line(actual) != clean_line(expected):
                      return False
    return True


def CompareFileContent(output, expect, filename):
    """Compare two files.  Return True if they match."""
    if filename in ['_perf', '_uuid', '_versions', '_log']:
        return True  # we never really expect these files to match.
    if os.path.basename(filename) == '_jobinfo':
        return CompareJobinfo(output, expect, filename)
    elif os.path.basename(filename) == '_finalstate':
        return CompareFinalState(output, expect, filename)
    else:
        return CompareLines(output, expect, filename)


def CompareContent(output, expect, filename):
    """Check that two paths contain the same content if they are files.  Does
    not check anything about non-file objects."""
    if not os.path.isfile(os.path.join(expect, filename)):
        if os.path.isfile(os.path.join(output, filename)):
            sys.stderr.write('File should not exist: %s\n'
                             % os.path.join(output, filename))
            return False
    else:
        if not CompareFileContent(output, expect, filename):
            sys.stderr.write('File content mismatch: %s\n' % filename)
            return False
    return True


def CheckResult(output_dir, expectation_dir, config):
    """Given an output directory and an expected output directory, and a config
    file containing the tests to apply, checks that the configured success
    criteria all pass."""
    ok = True
    if 'contains_files' in config:
        for pat in config['contains_files']:
            for fn in ExpandGlob(expectation_dir, pat):
                ok = CheckExists(output_dir, expectation_dir, fn) and ok
    if 'contains_only_files' in config:
        for pat in config['contains_only_files']:
            for fn in ExpandGlob(expectation_dir, pat):
                ok = CheckExists(output_dir, expectation_dir, fn) and ok
            for fn in ExpandGlob(output_dir, pat):
                ok = CheckExists(expectation_dir, output_dir, fn) and ok
    if 'contents_match' in config:
        for pat in config['contents_match']:
            for fn in ExpandGlob(expectation_dir, pat):
                ok = CompareContent(output_dir, expectation_dir, fn) and ok
    return ok


def main(argv):
    parser = optparse.OptionParser(usage='usage: %prog [options] <config>')
    options, argv = parser.parse_args(argv)
    if len(argv) != 2:
        parser.print_help()
        return 1
    with open(argv[1], 'r') as configfile:
        config = json.load(configfile)
    if not 'command' in config or not config['command']:
        sys.stderr.write('No command specified in %s\n' % argv[1])
    if not 'work_dir' in config:
        config['work_dir'] = os.path.dirname(argv[1])
    config['work_dir'] = os.path.abspath(config['work_dir'])
    config['command'][0] = os.path.abspath(os.path.join(
        os.path.dirname(argv[1]), config['command'][0]))
    output_dir = OutputDir(config)
    if output_dir and os.path.isdir(output_dir):
        shutil.rmtree(output_dir)
    sys.stderr.write("Running %s in %s.\n" %
                     (' '.join(config['command']), config['work_dir']))
    return_code = subprocess.call(config['command'], cwd=config['work_dir'])
    if 'expected_return' in config:
        if return_code != config['expected_return']:
            sys.stderr.write('Command returned %d\n' % return_code)
            return 2
    elif return_code != 0:
        sys.stderr.write('Command returned %d\n' % return_code)
        return 2
    expectation_dir = ExpectationDir(argv[1], config)
    if output_dir and expectation_dir:
        correct = CheckResult(output_dir, expectation_dir, config)
        if correct:
            sys.stderr.write('Output correct.\n')
            return 0
        else:
            sys.stderr.write('Output incorrect!\n')
            return 3


if __name__ == '__main__':
    sys.exit(main(sys.argv))
