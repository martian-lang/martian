#!/usr/bin/env python
#
# Copyright (c) 2016 10x Genomics, Inc. All rights reserved.
#

"""Script to run a test script and compare the output to an expected result.

Includes logic to ignore differences we expect from pipeline outputs,
such as timestamps, versions, and perf information.

"""

import itertools
import json
import optparse
import os
import re
import shutil
import subprocess
import sys

from fnmatch import fnmatchcase


def get_expectation_dir(config_filename, config):
    """Gets the absolute path of the 'expected output' directory from the
    config file."""
    if 'expected_dir' in config:
        return os.path.abspath(os.path.join(os.path.dirname(config_filename),
                                            config['expected_dir']))
    return os.path.abspath(os.path.join(os.path.dirname(config_filename),
                                        'expected'))


def get_output_dir(config):
    """Computes the absolute path of the test pipeline work directory from the
    config."""
    if 'output_dir' in config:
        return os.path.abspath(os.path.join(config['work_dir'],
                                            config['output_dir']))
    return None


def expand_glob(root, pattern):
    """Finds all of the files and directories matching pattern, relative to
    root.

    For example, if root is /mnt/awesome and pattern is 'foo/*/_outs' it might
    return foo/bar/_outs, foo/baz/_outs, foo/bar/baz/_outs and so on.

    This would be unnessessary in python3 because glob understands the
    ** recursive wildcard syntax, but in python 2 it is needed.

    """
    for cur, dirnames, filenames in os.walk(root):
        for name in filenames:
            if fnmatchcase(os.path.relpath(os.path.join(cur, name), root), pattern):
                yield os.path.relpath(os.path.join(cur, name), root)
        for name in dirnames:
            if fnmatchcase(os.path.relpath(os.path.join(cur, name), root), pattern):
                yield os.path.relpath(os.path.join(cur, name), root)


_UNIQUIFIER_REGEX = re.compile(
    '(.*%s[a-z]+[0-9]*)-u[0-9a-z]+(%s.+)' % (os.path.sep, os.path.sep))


def deuniquify(value):
    """Remove absolute paths and timestamps."""
    def uniquerepl(match):
        """Just take the matched group."""
        return '%s%s' % (match.group(1), match.group(2))
    return _UNIQUIFIER_REGEX.sub(uniquerepl, value)


_FILES_SEP = '%sfiles%s' % (os.path.sep, os.path.sep)


def _parent_files(value):
    """Convert file names like .../chnk0/files/foo to .../files/foo."""
    if not _FILES_SEP in value:
        return value
    if not os.path.basename(value) or not os.path.dirname(value):
        return value
    return os.path.join(
        os.path.dirname(os.path.dirname(os.path.dirname(value))),
        'files',
        os.path.basename(value))


def report_missing(item_type, output, expect, filename, reverse=False):
    """Print a message indicating a missing file."""
    if reverse:
        sys.stderr.write('Extra %s %s\n'
                         % (item_type, os.path.join(expect, filename)))
    else:
        sys.stderr.write('Missing %s %s\n'
                         % (item_type, os.path.join(output, filename)))


def check_exists_file(output, expect, filename, reverse=False):
    """Checks if a file exists in the output directory."""
    if reverse:
        filename = deuniquify(filename)
    if not os.path.isfile(os.path.join(output, filename)):
        if reverse:
            parent_filename = _parent_files(filename)
            if os.path.isfile(os.path.join(expect, parent_filename)):
                if not os.path.isfile(os.path.join(output, parent_filename)):
                    report_missing('file', output, expect,
                                   parent_filename, reverse)
                    return False
            else:
                report_missing('file', output, expect, filename, reverse)
                return False
        else:
            report_missing('file', output, expect, filename, reverse)
            return False
    return True


def check_exists(output, expect, filename, reverse=False):
    """Checks that a given file, directory, or link in the expected directory
    also exists in the output directory."""
    if os.path.basename(filename).startswith('.nfs'):
        return True  # These are temporary files created by nfs.
    if os.path.isdir(os.path.join(expect, filename)):
        return True  # git does not preserve empty directories
    if os.path.isfile(os.path.join(expect, filename)):
        return check_exists_file(output, expect, filename, reverse)
    elif os.path.islink(os.path.join(expect, filename)):
        if not os.path.islink(os.path.join(output, filename)):
            report_missing('link', output, expect, filename, reverse)
            return False
    else:
        raise Exception('No file %s'
                        % os.path.join(expect, filename))
    return True


def compare_dicts(actual, expected, keys):
    """Compares selected keys from two dictionaries."""
    if not actual:
        return not expected
    if not expected:
        return not actual
    for key in keys:
        if key in actual:
            if key in expected:
                return compare_objects(actual[key], expected[key])
            sys.stderr.write('Missing key %s\n' % key)
            return False
        elif key in expected:
            sys.stderr.write('Extra key %s\n' % key)
            return False
    return True


def compare_objects(actual, expected):
    """Compares two objects."""
    if not isinstance(actual, type(expected)):
        sys.stderr.write('Different types: %s != %s\n' %
                         (type(actual),
                          type(expected)))
        return False
    elif (isinstance(actual, unicode) and
          isinstance(expected, unicode) or
          isinstance(actual, str) and
          isinstance(expected, str)):
        if (clean_value(actual) !=
                clean_value(expected)):
            sys.stderr.write('Different strings: %s != %s\n' %
                             (clean_value(actual),
                              clean_value(expected)))
            return False
    elif isinstance(actual, dict):
        return (compare_dicts(actual, expected, actual.keys()) and
                compare_dicts(actual, expected, expected.keys()))
    elif (isinstance(actual, list) and
          isinstance(expected, list)):
        for actual_item, expected_item in itertools.izip_longest(
                sorted(actual), sorted(expected)):
            if not compare_objects(actual_item, expected_item):
                return False
    elif actual != expected:
        sys.stderr.write('%s != %s\n' %
                         (actual, expected))
        return False
    return True


def load_json(output, expect, filename):
    """Get two objects to compare from json."""
    try:
        with open(os.path.join(output, filename)) as act:
            actual = json.load(act)
        with open(os.path.join(expect, filename)) as exp:
            expected = json.load(exp)
    except IOError as err:
        sys.stderr.write('Error reading %s: %s\n' % (filename, err))
        return None, None, False
    except ValueError as err:
        sys.stderr.write('%s was not valid json: %s\n' % (filename, err))
        return None, None, False
    except TypeError as err:
        sys.stderr.write('%s contained invalid json types: %s\n' %
                         (filename, err))
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
    return compare_json(output, expect, filename, ['name',
                                                   'threads',
                                                   'memGB',
                                                   'type',
                                                  ])


def compare_final_state(output, expect, filename):
    """Compare two _finalstate json files.  Only compares keys within each
    element which are expected to remain the same across runs.

    In particular, we do not compare path, metadata, sweepbindings, or
    forks, all of which may contain absolute paths.

    """
    actual, expected, loaded = load_json(output, expect, filename)
    if not loaded:
        return False
    for actual_info, expected_info in itertools.izip_longest(actual, expected):
        if not compare_dicts(actual_info, expected_info, ['name',
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


_QUOTED_PATH_REGEX = re.compile('"/.*/([^/]+)"')
_PATH_REGEX = re.compile('^/.*/([^/]+)$')


def clean_value(value):
    """Remove absolute paths and timestamps."""
    def pathrepl(match):
        """Just take the matched group."""
        return '%s' % match.group(1)
    return _TIMESTAMP_REGEX.sub('__TIMESTAMP__',
                                _PATH_REGEX.sub(pathrepl, value))


def clean_line(line):
    """Remove absolute paths and timestamps."""
    def pathrepl(match):
        """Just take the matched group."""
        return '"%s"' % match.group(1)
    return _TIMESTAMP_REGEX.sub('__TIMESTAMP__',
                                _QUOTED_PATH_REGEX.sub(pathrepl, line))


def compare_lines(output, expect, filename):
    """Compare two files, replacing everything that might be an absolute path
    with the base path, and timestamps with __TIMESTAMP__."""
    with open(os.path.join(output, filename)) as act:
        with open(os.path.join(expect, filename)) as exp:
            for actual, expected in itertools.izip_longest(act, exp):
                if actual and expected:
                    if clean_line(actual) != clean_line(expected):
                        sys.stderr.write(
                            'Expected:\n%s\nActual:\n%s\n' %
                            (clean_line(expected), clean_line(actual)))
                        return False
    return True


def compare_file_content(output, expect, filename):
    """Compare two files.

    Return True if they match.

    """
    if filename in ['_perf', '_uuid', '_versions', '_log']:
        return True  # we never really expect these files to match.
    if os.path.basename(filename) == '_jobinfo':
        return compare_jobinfo(output, expect, filename)
    elif os.path.basename(filename) == '_finalstate':
        return compare_final_state(output, expect, filename)
    elif os.path.basename(filename) in ['_outs',
                                        '_args',
                                        '_stage_defs',
                                        '_vdrkill.partial',
                                        '_vdrkill']:
        return compare_json(output, expect, filename)
    return compare_lines(output, expect, filename)


def compare_content(output, expect, filename):
    """Check that two paths contain the same content if they are files.

    Does not check anything about non-file objects.

    """
    if not os.path.isfile(os.path.join(expect, filename)):
        if os.path.isfile(os.path.join(output, filename)):
            sys.stderr.write('File should not exist: %s\n'
                             % os.path.join(output, filename))
            return False
    elif not os.path.isfile(os.path.join(output, filename)):
        sys.stderr.write('Missing file %s\n'
                         % os.path.join(output, filename))
        return False
    else:
        if not compare_file_content(output, expect, filename):
            sys.stderr.write(
                'File content mismatch: %s\n'
                'Expected content: %s\n'
                'Actual content: %s\n' % (filename,
                                          os.path.join(expect, filename),
                                          os.path.join(output, filename)))
            return False
    return True


def check_result(output_dir, expectation_dir, config):
    """Given an output directory and an expected output directory, and a config
    file containing the tests to apply, checks that the configured success
    criteria all pass."""
    result_ok = True
    if 'contains_files' in config:
        for pat in config['contains_files']:
            for fname in expand_glob(expectation_dir, pat):
                result_ok = check_exists(
                    output_dir, expectation_dir, fname) and result_ok
    if 'contains_only_files' in config:
        for pat in config['contains_only_files']:
            for fname in expand_glob(expectation_dir, pat):
                result_ok = check_exists(
                    output_dir, expectation_dir, fname) and result_ok
            for fname in expand_glob(output_dir, pat):
                result_ok = check_exists(
                    expectation_dir, output_dir, fname,
                    reverse=True) and result_ok
    if 'contents_match' in config:
        for pat in config['contents_match']:
            for fname in expand_glob(expectation_dir, pat):
                result_ok = compare_content(
                    output_dir, expectation_dir, fname) and result_ok
    return result_ok


def main(argv):
    """Execute the test case."""
    parser = optparse.OptionParser(usage='usage: %prog [options] <config>')
    _, argv = parser.parse_args(argv)
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
    output_dir = get_output_dir(config)
    if output_dir and os.path.isdir(output_dir):
        shutil.rmtree(output_dir)
    sys.stderr.write('Running %s in %s.\n' %
                     (' '.join(config['command']), config['work_dir']))
    return_code = subprocess.call(config['command'], cwd=config['work_dir'])
    if 'expected_return' in config:
        if return_code != config['expected_return']:
            sys.stderr.write('Command returned %d\n' % return_code)
            return 2
    elif return_code != 0:
        sys.stderr.write('Command returned %d\n' % return_code)
        return 2
    expectation_dir = get_expectation_dir(argv[1], config)
    if output_dir and expectation_dir:
        correct = check_result(output_dir, expectation_dir, config)
        if correct:
            sys.stderr.write('Output correct.\n')
            return 0
        sys.stderr.write('Output incorrect!\n')
        return 3
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv))
