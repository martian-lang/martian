Before submitting an issue, be sure to read our [contribution guidelines](CONTRIBUTING.md)

## Component

[e.g. `mrp`, `mro`, as well as the repository revision or release tag]

## Pipeline

Please specify one of

* A released 10X genomics pipeline or one of the test pipelines in this repository.
* An open-source repository containing mro and stage code.
* A simple mro file, along with stage code and directory layout.

## Runtime environment

Include the full command line used to run the executable.

Include all non-sensitive environment variables.  We don't want to see user or host names.

Include the operating system distribution and version (include the output of `uname -a`).

Especially for cluster mode jobs, include any relevant configuration files,
especially the job template in the `jobmanagers`.

For MRP issues, specify the filesystem you are running on (e.g. ext3, nfs backed by zfs, gpfs) if you know.

Much of this information and more is in the `sitecheck.txt` produced by 10X pipelines.

## Expected behavior

What did you want to happen?

## Actual behavior

For `mrp` failures, please include if possible:

* If it was produced, and does not contain any sensitive information, attach the `[id].mri.tgz`.
* Otherwise:
  * The pipestance `_log` file
  * The output of `find [pipestance_directory] -name _\* -type f`
  * For failed stages, at least the `_errors`, `_log`, `_stdout`, and `_stderr` files.
