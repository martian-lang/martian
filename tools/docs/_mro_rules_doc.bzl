"""Rules for dealing with [Martian](https://martian-lang.org) `.mro` sources."""

# Do not include this file - it is intended for generating documentation.
# Use mro_rules.bzl instead.

load("//tools/private:mro_library.bzl", _mro_library = "mro_library")
load("//tools/private:mro_test.bzl", _mro_test = "mro_test")
load(
    "//tools/private:mrf_rules.bzl",
    _mrf_test = "mrf_test",
    _mro_tool_runner = "mro_tool_runner",
)
load(
    "//tools:mro_rules.bzl",
    _mrf_runner = "mrf_runner",
)

mro_library = _mro_library

mro_test = _mro_test

mro_tool_runner = _mro_tool_runner

mrf_runner = _mrf_runner

mrf_test = _mrf_test
