martian
=======

[![Build Status](https://travis-ci.org/martian-lang/martian.svg?branch=master)](https://travis-ci.org/martian-lang/martian)

How to Clone Me
---------------
This repo includes vendored third-party code as submodules, so it must be git cloned recursively:

```
> git clone https://github.com/martian-lang/martian.git --recursive
```

Martian Executables
-------------------
To view commandline usage and options for any executable, give the `--help` option.

- `mrc` Commandline MRO compiler. Checks syntax and semantics of MRO files.
- `mrf` Commandline canonical code formatter for MRO files.
- `mrp` Pipeline runner.
- `mrs` Single-stage runner.

