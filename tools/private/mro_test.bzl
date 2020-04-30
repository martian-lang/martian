"""Rule for checking martian mro source files."""

load("//tools:providers.bzl", "MroInfo")

def _run_mrc(mro, mropath, mrofiles, flags):
    return """#!/usr/bin/env bash
export MROPATH="{}"
echo $MROPATH
exec -a {} "{}" check{} {}
""".format(
        mropath,
        mro.basename,
        mro.short_path,
        " " + " ".join(flags) if flags else "",
        " ".join([
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
