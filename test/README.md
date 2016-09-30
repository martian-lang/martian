# Martian testing framework

This is a simple framework for writing integration tests for the Martian runtime.

## Running a test
```bash
$ ./martian_test.py test_spec.json
```

`test_spec.json` defines the command to run as well as expected results.

## Spec JSON format
```json
{
  "command": ["my_script.sh", "arg1", "arg2"],
  "work_dir": "/path/to/script",
  "expected_return": 0,
  "output_dir": "/path/to/outputs",
  "expected_dir": "/path/to/expected/outputs",
  "contains_files": ["*"],
  "contains_only_files": ["outs/*"],
  "contents_match": ["outs/fork0/output.json", "outs/fork1/*"]
}
```

<table>
<tr><th> Argument  </th><th> Required </th><th>     </th></tr><tr><td> command </td><td> yes      </td><td> The command to run, or a list containing the command
and its arguments.  A list is preferred, e.g. ["ls", "-al", path]. </td></tr>
<tr><td> work_dir </td><td> no      </td><td> The working directory where the command should be   run.
The default behavior is to run in the directory containing the config. </td></tr>
<tr><td> expected_return </td><td> no </td><td> The expected return code for the command.
The default is 0. </td></tr>
<tr><td> output_dir </td><td> no* </td><td> The root for the output directory, relative to the
working directory.  This directory will be deleted before running the command.
All of the subsequent arguments are ignored if no output directory is
specified. </td></tr>
<tr><td> expected_dir </td><td> no </td>
<td> The root directory for the "gold standard" truth files,
relative to the location of the config.  Default is 'expected' in the
directory containing the config file. </th></tr>
<tr><td> contains_files </td><td> no </td>
<td> A list of paths, relative to the expected_dir, for
which the test will fail if the files do not also exist in the same relative
 location in the output_dir.  Wildcards are treated recursively. </td></tr>
<tr><td> contains_only_files </td><td> no </td>
<td> A list of paths for which the test will fail if
the list of files in output_dir does not exactly match the list of files
in expected_dir. </td></tr>
<tr><td> contents_match </td><td> no </td>
<td> A list of files for which the test will fail if the
output file content is not an exact match for the content in the
 expected_dir. </td></tr>
</table>

## Content matching
Files specified in `contents_match` do not have to match exactly.
`_perf`, `_uuid`, `_versions`, and `_log` in the pipestance root are
automatically ignored.  `_jobinfo` and `_finalstate` have selected keys
compared only.  All other files are compared ignoring absolute paths
(only looking at the base file name) and timestamps.

## Creating a test case
### Set Up the test script
First create the MRO and test script to run in.  The test script should set
up any environment variables you depend on, as when the tests run in Travis
they are run with a clean environment.  If your test can run as a single
command with no setup, you do not need a script and can specify the test
entirely within the json file.

Most common tests will have a json configuration that looks like
```json
{
  "command": ["run_test.sh"],
  "output_dir": "pipeline_test",
  "contains_only_files": ["*"],
  "contents_match": ["*"]
}
```
and `run_test.sh` looks like
```bash
#!/bin/bash
MROPATH=$PWD
PATH=../../bin:$PATH
mrp pipeline.mro pipeline_test
```

### Create the expected output directory
Next, run the script from the intended working directory, as specified by
the json file (see above).  Move the output pipestance directory to the
location specified by `expected_dir` in the json, and if desired run
something like
```bash
$ find | xargs perl -i -pe 's/\/old\/abs\/path/\/friendly\/abs\/path/g'
```
in order to get your username out of files which are getting committed.

If the expected data involves a large number of files, you may want to tar
it and have the test script untar it.  If it is large enough to justify
compression, store it with [git lsf](https://git-lfs.github.com/) rather than
committing directly to git.  Keep in mind that for short-running tests that
will be running in Travis, the test cannot depend on resources which are not
publicly available such as cluster NFS directories.

### Test your test
Ensure you have configured the test correctly by running it in a clean
environment on a linux machine.
