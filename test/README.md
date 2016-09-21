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
