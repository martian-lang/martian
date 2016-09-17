#!/usr/bin/env python

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
    if 'expected_dir' in config:
        return os.path.abspath(os.path.join(os.path.dirname(config_filename),
                                            config['expected_dir']))
    else:
        return os.path.abspath(os.path.join(os.path.dirname(config_filename),
                                            'expected'))


def OutputDir(config):
    if 'output_dir' in config:
        return os.path.abspath(os.path.join(config['work_dir'],
                                            config['output_dir']))
    else:
        return None


def ExpandGlob(root, pattern):
    for cur, dirnames, filenames in os.walk(root):
        for fn in filenames:
            if fnmatchcase(os.path.relpath(os.path.join(cur, fn), root), pattern):
                yield os.path.relpath(os.path.join(cur, fn), root)
        for fn in dirnames:
            if fnmatchcase(os.path.relpath(os.path.join(cur, fn), root), pattern):
                yield os.path.relpath(os.path.join(cur, fn), root)


def CheckExists(output, expect, filename):
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


def CompareTimestamped(output, expect, filename):
    with open(os.path.join(output, filename)) as act:
        with open(os.path.join(expect, filename)) as exp:
            for actual, expected in itertools.izip_longest(act, exp):
                if actual and expected:
                    if (_TIMESTAMP_REGEX.sub(actual, '__TIMESTAMP__') !=
                            _TIMESTAMP_REGEX.sub(expected, '__TIMESTAMP__')):
                        return False
    return True


def CompareFileContent(output, expect, filename):
    if os.path.basename(filename) == '_jobinfo':
        return CompareJobinfo(output, expect, filename)
    elif os.path.basename(filename) == '_finalstate':
        return CompareFinalState(output, expect, filename)
    elif os.path.basename(filename) in ['_complete',
                                        '_vdrkill',
                                        '_log',
                                       ]:
        return CompareTimestamped(output, expect, filename)
    else:
        return filecmp.cmp(os.path.join(output, filename),
                           os.path.join(expect, filename))


def CompareContent(output, expect, filename):
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
    if 'expected_return' in config and return_code != config['expected_return']:
        sys.stderr.write('Command returned %d\n' % return_code)
        return 2
    elif return_code != 0:
        sys.stderr.write('Command returned %d\n' % return_code)
        return 2
    expectation_dir = ExpectationDir(argv[1], config)
    if output_dir and expectation_dir:
        correct = CheckResult(output_dir, expectation_dir, config)
        if correct:
            sys.stderr.write('Output correct.')
            return 0
        else:
            sys.stderr.write('Output incorrect!')
            return 3


if __name__ == '__main__':
    sys.exit(main(sys.argv))
