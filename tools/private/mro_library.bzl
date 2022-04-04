"""Rule for martian mro source files."""

load("//tools:providers.bzl", "MroInfo")
load("//tools:util.bzl", "merge_pyinfo")
load("@bazel_skylib//lib:paths.bzl", "paths")

def _mro_library_impl(ctx):
    rf = ctx.runfiles(
        files = ctx.files.srcs + ctx.files.data,
        collect_data = True,
        collect_default = True,
    )

    if ctx.attr.mropath:
        mropath = [
            paths.normalize(paths.join(f.dirname, ctx.attr.mropath))
            for f in ctx.files.srcs
        ]
    else:
        mropath = []
    stage_py_deps = [
        dep[DefaultInfo].files
        for dep in ctx.attr.deps
        if MroInfo not in dep and PyInfo in dep
    ]
    return [
        MroInfo(
            mropath = depset(
                mropath,
                order = "preorder",
                transitive = [
                    dep[MroInfo].mropath
                    for dep in ctx.attr.deps
                    if MroInfo in dep
                ],
            ),
            transitive_mros = depset(
                ctx.files.srcs,
                transitive = [
                    dep[MroInfo].transitive_mros
                    for dep in ctx.attr.deps
                    if MroInfo in dep
                ],
            ),
            stage_py_deps = depset(
                transitive = stage_py_deps + [
                    dep[MroInfo].stage_py_deps
                    for dep in ctx.attr.deps
                    if MroInfo in dep
                ],
            ),
        ),
        DefaultInfo(files = depset(ctx.files.srcs), runfiles = rf),
        merge_pyinfo(deps = [dep for dep in ctx.attr.deps if PyInfo in dep]),
    ]

mro_library = rule(
    attrs = {
        "srcs": attr.label_list(
            allow_files = [".mro"],
            doc = ".mro source file(s) for this library target.",
        ),
        "data": attr.label_list(
            allow_files = True,
            doc = "Data dependencies for top-level calls defined in srcs.",
        ),
        "deps": attr.label_list(
            providers = [
                [PyInfo],
                [MroInfo],
                [DefaultInfo],
            ],
            doc = "Included mro libraries and stage code for stages defined in srcs.",
        ),
        "mropath": attr.string(
            default = ".",
            doc = """A path to add to the `MROPATH` for targets which
depend on this `mro_library`, relative to the package.

The current default is '.', meaning the package directory.  This will change
in the future, after some time for migration, to be empty instead.""",
        ),
    },
    doc = """
A rule for collecting an mro file and its dependencies.

Transitively collects `MROPATH` from other `mro_library` dependencies,
as well as `PYTHONPATH` from any python dependencies.

`.mro` files should be supplied in `srcs`.  Other `mro` targets included
by files in `srcs`, as well as the targets implementing stages declared in the
`srcs`, should be included in `deps`.
""",
    provides = [MroInfo],
    implementation = _mro_library_impl,
)
