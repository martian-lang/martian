# Contributions

Martian was originally created at
[10x Genomics](https://www.10xgenomics.com/). We are excited to
now invite the community to contribute to its development!

## Contributor License Agreement
Martian is released under the [MIT License](../LICENSE).  By contributing to
the project, including but not limited to through issues and pull requests,
you are agreeing to release any intellectual property contained in those
contributions under the same terms.  We cannot accept contributions which
impose other licensing terms on the project.

## Supported Platforms
Martian is intended to run primarily on `x86_64` Linux platforms. We target
support for running on the most recent patch versions of CentOS 5.2 or
greater and Ubuntu 10 or greater.

We'd like if it built on OSX and ideally Windows, and if at least some of the
tools worked on those platforms.

Martian is written primarily in [Go](https://golang.org/).  Go's
[release policy](https://golang.org/doc/devel/release.html#policy) specifies
that only the most recent point revision is supported.  We do not expect
to be able to build with unsupported versions of Go.

The Martian Python adapter is written for Python 2.7.  We have not tested with
Python 3.x, but fixes are welcome.

The web user interface is built using [Node](https://nodejs.org/) 6, which is
the most recent LTS version.  Any fixes for newer versions are welcome,
so long as they are backwards compatible.

## Release Policy
For the open source Martian repository, only the most recent (non-`rc`)
release is supported.  

Development happens on the master branch, and the intent is for the master
branch to always be in a "releasable" state.  Features will not be back-ported
to previous release branches.  Pull requests for bug fixes may be cherry-picked
to a previous release branches in some circumstances, but the normal action
would be to create a new release off of master.

## Issues
If you are running a released 10X Genomics pipeline, such as
longranger or cellranger, with a released version of Martian (packaged
in the release tarballs under `martian-cs/<version>/`), please contact
[10X support](https://support.10xgenomics.com/) for assistance.  Running
a pipeline with a version of Martian you built yourself is not supported
by 10x Genomics.

Before submitting a bug, check to make sure it is not a duplicate of an
existing issue.  Also, check the
[documentation](http://martian-lang.org/) to ensure your environment is set up
correctly.  Also, verify that your repository is up to date, including
submodules, in case the issue was already fixed.

For issues with `mrp`, see if you can reproduce the issue
with a known-working pipeline (such as the ones in the [`test/`](../test)
directory or a released 10x pipeline).  If an `[id].mri.tgz` was produced, include
that if possible (keeping in mind that GitHub issues are public.  Make sure you
are not uploading anything you don't want the world to see!).

Local mode and cluster mode with SGE are expected to work in most
configurations. Other cluster managers are less well tested, but we would love
to see contributions to improve support for them.

For issues with documentation, please file them in the
[documentation](https://github.com/martian-lang/martian-docs) repository.

## Patch Acceptance Process
We use normal GitHub pull requests for patches.  Before sending a pull request,
please do the following:

1. Make sure there is a bug or feature request associated with your change, and
mention it in the pull request description.  This allows for a separation
between discussion of what should be added or fixed and how it is implemented.
2. For new features, allow for a discussion on the issue so everyone can get on
board with the feature design and point out any potential pitfalls before going
to the work of implementing it.
3. For bug fixes, if possible create a unit test case which reproduces the bug
so that we can prevent regressions.
4. Make sure your change is based on the current `master` branch HEAD revision.
Patches to previous releases are accepted, but our normal process is to fix
on master and cherry-pick to the release branch.
4. For Go code, make sure it is run through
[`gofmt`](https://golang.org/cmd/gofmt/) and
[`go vet`](https://golang.org/cmd/vet/).  For python code, use `pyformat` and
`pylint` before submitting.  These tools prevent distracting churn in
formatting and provide a first line of defense against error-prone code.
5. Ideally, write a unit test or integration test to cover your new code. Test
coverage is poor right now, but we'd like to improve it.  Test-only Pull
Requests are very welcome!
6. Make sure your change builds and passes the basic integration tests.  Run
`make && make longtests`.
7. Wait for a reviewer to be assigned to the pull request.  We're a small team,
so it might take a week or so.
8. Complete the review process.  This may require some back and forth with the
reviewer.  Depending on how long this takes, it may be necessary to
periodically merge changes from master into the pull request branch.

## Patch Priorities
Martian is under active development internally, so many things you might want
are already planned!  See the [roadmap](http://martian-lang.org/roadmap) for
our current plans.  That said, we have limited resources, and your favorite
feature request might not be something we can prioritize.  Pull requests are
especially welcome for the following:

* Cluster mode improvements, particularly on non-SGE clusters.  We have an
SGE cluster, but we do not yet have a Slurm or LSF cluster to test on.  Those
cluster managers are increasing in popularity, so we'd love to see improvements
to support for them, or other cluster managers which we haven't heard of. In
particular, additions or improvements to the
[`jobmanagers/config.json`](../jobmanagers/config.json) and template examples,
or implementation of queue-querying scripts like `sge_queue.py` for other
cluster types.
* Support for more stage code languages.  For compiled languages other than Go,
it's best to keep those in a separate repository as we did for
[Rust](https://github.com/martian-lang/martian-rust), since they cannot and
should not be tightly coupled to a specific version of Martian.  For scripting
languages such as Python, however, a
[front-end wrapper adapter](../adapters/python/martian_shell.py) may simplify
stage code development.  Be sure to include additions to the mro
[lexer](../src/martian/core/lexer.go) and [grammar](../src/martian/syntax/grammar.y)
to support the new mode.  Be sure to read the
[API documentation](../adapters/README.md) before implementing an adapter.
* Improvements for Python 3 compatibility are welcome.  We've done no testing
at all with Python 3.
* Editor support for mro.  Currently we have [editor plugins](../tools/syntax)
for Vim, Atom, and Sublime editors.
