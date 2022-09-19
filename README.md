<p align="center">
  <a href="http://martian-lang.org">
    <img src="https://avatars0.githubusercontent.com/u/16513506?v=4&s=200">
  </a>
  <p align="center">
    Language and framework for developing high performance computational pipelines.
  </p>
</p>

[![GoDoc](https://godoc.org/github.com/martian-lang/martian?status.svg)](https://godoc.org/github.com/martian-lang/martian)
[![Build Status](https://github.com/martian-lang/martian/actions/workflows/test.yml/badge.svg)](https://github.com/martian-lang/martian/actions/workflows/test.yml)

## Getting Started

Please see the [Martian Documentation](http://martian-lang.org).

The easiest way to get started is

```sh
$ git clone https://github.com/martian-lang/martian.git
$ cd martian
$ make
```

Alternatively, build with [bazel](https://bazel.build):

```sh
$ bazel build //:mrp
```

Note that while `go get` will not work very well for this repository, because it
will skip fetching the web UI and runtime configuration files, and because this
repository had already tagged version 3 by the time go modules came around, and
is thus not following the expected go conventions for how code is organized for
non-v1 versions.

### Note on semantic versioning

Semantic versioning for martian is based on pipeline compatibility, not the Go
API.  That is, a major version change indicates that pipelines (defined in
`.mro` files) may no longer function correctly.  This unfortunately poses problems
with go modules, (which didn't exist yet at the time v3 was first tagged) which
expect version tags to be referring to the semver compatibility of the Go API.
We hope to rectify this in v5, but ironically that will force an
backwards-incompatible change to the Go API.  In the mean time, if you wish to
depend on martian as a go module, use the git commit rather than a version tag.

## Copyright and License

Code and documentation copyright 2014-2017 the [Martian Authors](https://github.com/martian-lang/martian/graphs/contributors) and [10x Genomics, Inc.](https://10xgenomics.com) Code released under the [MIT License](https://github.com/martian-lang/martian/blob/master/LICENSE). Documentation released under [Creative Commons](https://github.com/martian-lang/martian-docs/blob/master/LICENSE).
