"""Rules for running `mro format`, either to reformat files or to check them."""

load(
    "//tools:providers.bzl",
    "MroInfo",
)
load("//tools:util.bzl", "merge_runfiles")
load("@bazel_skylib//lib:shell.bzl", "shell")

def _translate_bin_path(p):
    if p.startswith(".."):
        return "external/" + p[len("../"):]
    return p

def _mro_tool_common_impl(ctx, script_function):
    mros = depset(
        ctx.files.srcs,
        transitive = [
            dep[MroInfo].transitive_mros
            for dep in ctx.attr.srcs
            if MroInfo in dep
        ],
    ).to_list()
    script = ctx.actions.declare_file(ctx.attr.script_name or ctx.attr.name)
    ctx.actions.write(
        output = script,
        content = script_function(ctx, mros),
        is_executable = True,
    )
    mros.append(script)
    return [
        DefaultInfo(
            executable = script,
            files = depset([script]),
            runfiles = merge_runfiles(
                ctx,
                [ctx.attr._mro],
                files = mros,
            ),
        ),
    ]

def _make_mrf_runner_script(ctx, mros):
    return """#!/usr/bin/env bash
root=$(dirname $(realpath -sL "${{BASH_SOURCE[0]}}"))
runfiles="${{root}}"
if [ -d "${{BASH_SOURCE[0]}}.runfiles/{workspace}" ]; then
    runfiles=$(realpath -sL "${{BASH_SOURCE[0]}}.runfiles/{workspace}")
fi
if [ -d "${{BUILD_WORKING_DIRECTORY}}" ]; then
    builtin cd "${{BUILD_WORKING_DIRECTORY}}"
fi
{mropath}
export MARTIAN_BASE=$(dirname "${{runfiles}}/{mro}")
exec -a {basename} "${{runfiles}}/{mro}" {subcommand}{flags} \\
\t{mro_files}
""".format(
        basename = shell.quote(ctx.executable._mro.basename),
        mropath = "export MROPATH=\"{}\"".format(":".join([
            "${runfiles}/" + _translate_bin_path(p)
            for p in depset(transitive = [
                dep[MroInfo].mropath
                for dep in ctx.attr.srcs
                if MroInfo in dep
            ]).to_list()
        ])) if ctx.attr.srcs else "",
        mro = _translate_bin_path(ctx.executable._mro.short_path),
        subcommand = shell.quote(ctx.attr.subcommand),
        flags = " \\\n\t" + " \\\n\t".join(
            [
                shell.quote(flag)
                for flag in ctx.attr.flags
            ],
        ) if ctx.attr.flags else "",
        mro_files = " \\\n\t".join([
            "\"${runfiles}/\"" + shell.quote(_translate_bin_path(p.short_path))
            for p in mros
        ]) + " \"$@\"",
        workspace = ctx.workspace_name or "__main__",
    )

def _mro_tool_runner_impl(ctx):
    return _mro_tool_common_impl(ctx, _make_mrf_runner_script)

mro_tool_runner = rule(
    attrs = {
        "srcs": attr.label_list(
            doc = "The mro files to give as arguments to the tool.",
            allow_files = True,
            providers = [
                [MroInfo],
                [DefaultInfo],
            ],
        ),
        "subcommand": attr.string(
            mandatory = True,
            doc = "The subcommand for the `mro` tool, e.g. " +
                  "`format`, `check`, `graph`, `edit`.",
        ),
        "_mro": attr.label(
            executable = True,
            default = Label("@martian//:mro"),
            cfg = "target",
        ),
        "script_name": attr.string(
            doc = "The name for the script file.",
        ),
        "flags": attr.string_list(
            default = [],
            doc = "Flags to pass to the `mro` subcommand.",
        ),
    },
    doc = "Runs the `mro` tool, possibly with a subcommand, with the given " +
          "mro files (if any) as arguments.",
    executable = True,
    implementation = _mro_tool_runner_impl,
)

def _make_mrf_tester_script(ctx, mros):
    if not mros:
        fail("required: at least one file to check", "mros")
    return """#!/usr/bin/env bash
export MROPATH="{mropath}"
export MARTIAN_BASE=$(dirname "{mro}")
rc=0
for mro_file in \\
\t{mro_files}; do
    echo "checking ${{mro_file}}..."
    (
        set -o pipefail
        diff "${{mro_file}}" <("{mro}" format "${{mro_file}}")
    ) || rc=$?
done
exit $rc
""".format(
        mropath = ":".join(
            depset(transitive = [
                dep[MroInfo].mropath
                for dep in ctx.attr.srcs
                if MroInfo in dep
            ]).to_list(),
        ),
        mro = ctx.executable._mro.short_path,
        mrf_flags = " ".join(ctx.attr.mrf_flags),
        mro_files = " \\\n\t".join([
            shell.quote(p.short_path)
            for p in mros
        ]),
    )

def _mrf_test_impl(ctx):
    return _mro_tool_common_impl(ctx, _make_mrf_tester_script)

mrf_test = rule(
    attrs = {
        "srcs": attr.label_list(
            mandatory = True,
            doc = "The mro files to format.",
            allow_files = True,
            providers = [
                [MroInfo],
                [DefaultInfo],
            ],
        ),
        "_mro": attr.label(
            executable = True,
            default = Label("@martian//:mro"),
            cfg = "target",
        ),
        "script_name": attr.string(
            doc = "The name for the script file.",
        ),
        "mrf_flags": attr.string_list(
            default = [
                "--includes",
            ],
            doc = "Flags to pass to `mro format`.",
        ),
    },
    doc = "Runs `mro format` on the given files, and fails if there are any differences.",
    test = True,
    implementation = _mrf_test_impl,
)
