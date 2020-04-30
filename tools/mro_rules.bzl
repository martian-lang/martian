"""Rules for dealing with martian mro pipeline sources."""

load("//tools/private:mro_library.bzl", _mro_library = "mro_library")
load("//tools/private:mro_test.bzl", _mro_test = "mro_test")
load(
    "//tools/private:mrf_rules.bzl",
    _mrf_test = "mrf_test",
    _mro_tool_runner = "mro_tool_runner",
)

mro_library = _mro_library

def mro_test(size = "small", **kwargs):
    """Runs `mro check` on the given `mro_library` target.

    Args:
      size: See https://docs.bazel.build/be/common-definitions.html#test.size
      **kwargs: Other attributes to pass to the test rule.
    """
    _mro_test(size = size, **kwargs)

def mro_tool_runner(
        tags = [],
        **kwargs):
    """Run the `mro` tool without sandboxing.  See `_mro_tool_runner`.

    Args:
        tags: Any `tags` to pass besides `"local"`.  See
              [bazel docs](https://docs.bazel.build/be/common-definitions.html#common.tags).
        **kwargs: The remaining attributes for the rule.
    """
    _mro_tool_runner(
        tags = tags + ["local"],
        **kwargs
    )

def mrf_runner(
        mrf_flags = [
            "--rewrite",
            "--includes",
        ],
        srcs = [],
        **kwargs):
    """A convenience wrapper for running `mro format` on a set of files.

    Args:
      mrf_flags: Flags to pass to `mro format`.
      srcs: The set of mro files or targets to format.  If an `mro_library`
        target is supplied, its transitive dependencies will be formatted
        as well.
      **kwargs: Any other attributes to pass through to the underlying rule,
        including for example `name` and `visibility`.
    """
    mro_tool_runner(
        subcommand = "format",
        flags = kwargs.pop("flags", []) + mrf_flags,
        srcs = srcs,
        **kwargs
    )

def mrf_test(
        size = "small",
        **kwargs):
    """Wraps the `mro format` test, setting size="small" by default.  See `_mrf_test`.

    Args:
        size: Any `tags` to pass besides `"local"`.  See
              [bazel docs](https://docs.bazel.build/be/common-definitions.html#test.size).
        **kwargs: The remaining attributes for the rule.
    """
    _mrf_test(
        size = size,
        **kwargs
    )
