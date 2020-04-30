""" Bazel rule for running a pipeline integration test.

Intended for internal use in the martian repository.
"""

load("//tools:providers.bzl", "MroInfo")
load("//tools:util.bzl", "merge_runfiles")

def _integration_test_impl(ctx):
    script = ctx.actions.declare_file(ctx.attr.name)
    ctx.actions.write(script, """#!/usr/bin/env bash
if [ -d "${{BASH_SOURCE[0]}}.runfiles/{workspace}" ]; then
    builtin cd "${{BASH_SOURCE[0]}}.runfiles/{workspace}"
fi
export PATH="${{PWD}}/{mrp}:${{PATH}}"

exec "{tester}" "{config}"
""".format(
        workspace = ctx.workspace_name or "__main__",
        tester = ctx.executable._tester.short_path,
        mrp = ctx.executable._mrp.dirname,
        config = ctx.file.config.short_path,
    ), is_executable = True)
    runfiles = merge_runfiles(
        ctx,
        files = [script, ctx.file.config, ctx.file.runner] +
                ctx.files.expectation + ctx.files.data,
        deps = [
            ctx.attr.pipeline,
            ctx.attr._tester,
            ctx.attr._mrp,
        ],
    )
    return DefaultInfo(
        files = depset([script]),
        runfiles = runfiles,
        executable = script,
    )

integration_test = rule(
    attrs = {
        "config": attr.label(allow_single_file = [".json"]),
        "pipeline": attr.label(providers = [MroInfo]),
        "runner": attr.label(allow_single_file = True),
        "expectation": attr.label_list(allow_files = True),
        "data": attr.label_list(allow_files = True),
        "_tester": attr.label(
            executable = True,
            cfg = "target",
            default = "//test:martian_test",
        ),
        "_mrp": attr.label(
            executable = True,
            cfg = "target",
            default = "//:mrp",
        ),
    },
    implementation = _integration_test_impl,
    test = True,
    doc = "Runs a martian integration test.",
)
