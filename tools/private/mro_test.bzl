"""Rule for checking martian mro source files."""

load("//tools:providers.bzl", "MroInfo")

def _run_mrc(workspace, mro, mropath, mrofiles, flags):
    return """#!/usr/bin/env bash
if [[ -e "${{BASH_SOURCE[0]}}.runfiles/{workspace}" ]]; then
    echo "Running in ${{BASH_SOURCE[0]}}.runfiles/{workspace}"
    cd "${{BASH_SOURCE[0]}}.runfiles/{workspace}"
fi
export MROPATH="{mropath}"
echo $MROPATH
exec -a {basename} "{cmd}" check{strict} {args}
""".format(
        workspace = workspace,
        mropath = mropath,
        basename = mro.basename,
        cmd = mro.short_path,
        strict = " " + " ".join(flags) if flags else "",
        args = " ".join([
            "\"" + f.short_path + "\""
            for f in mrofiles
        ]),
    )

def _mro_test_impl(ctx):
    mro = ctx.executable._mro
    mropath = ":".join(depset(
        transitive = [
            src[MroInfo].mropath
            for src in ctx.attr.srcs
        ],
    ).to_list())
    flags = ctx.attr.flags
    if ctx.attr.strict and "--strict" not in flags:
        flags = flags + ["--strict"]
    script = _run_mrc(
        ctx.workspace_name or "__main__",
        mro,
        mropath,
        ctx.files.srcs,
        flags,
    )
    ctx.actions.write(
        output = ctx.outputs.executable,
        content = script,
    )
    runfiles = ctx.runfiles(
        files = [ctx.executable._mro],
        collect_data = True,
    )
    return [DefaultInfo(runfiles = runfiles)]

mro_test = rule(
    attrs = {
        "strict": attr.bool(
            doc = "Run `mro check` with the `--strict` flag.",
            default = True,
        ),
        "flags": attr.string_list(
            doc = "Additional flags to pass to `mro check`",
            default = [],
        ),
        "_mro": attr.label(
            default = "@martian//:mro",
            executable = True,
            cfg = "host",
        ),
        "srcs": attr.label_list(providers = [MroInfo]),
    },
    doc = """
Runs `mro check` on the `mro_library` targets supplied in `srcs`.
""",
    test = True,
    implementation = _mro_test_impl,
)
