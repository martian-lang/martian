"""Rule for generating go code with `mro2go`."""

load("//tools:providers.bzl", "MroInfo")

def _mro2go_codegen_impl(ctx):
    if ctx.attr.pipelines and ctx.attr.stages:
        fail("Both `pipelines` and `stages` were specified.")
    mropath = ":".join(depset(
        transitive = [
            src[MroInfo].mropath
            for src in ctx.attr.srcs
        ],
    ).to_list())
    transitive_srcs = depset(transitive = [
        src[MroInfo].transitive_mros
        for src in ctx.attr.srcs
    ])
    outs = [
        ctx.actions.declare_file(f.basename[:-1 - len(f.extension)] + ".go")
        for f in ctx.files.srcs
    ]
    if not outs:
        fail("Empty inputs.", attr = "srcs")
    args = ctx.actions.args()
    args.add("-output-dir", outs[0].dirname)
    if ctx.attr.inputs_only:
        args.add("-input-only")
    if ctx.attr.stages:
        args.add_joined("-stage", ctx.attr.stages, join_with = ",")
    if ctx.attr.pipelines:
        args.add_joined("-pipeline", ctx.attr.pipelines, join_with = ",")
    if ctx.attr.package:
        args.add("-package", ctx.attr.package)
    args.add_all(ctx.files.srcs)
    ctx.actions.run(
        executable = ctx.executable._mro2go,
        inputs = transitive_srcs,
        outputs = outs,
        arguments = [args],
        mnemonic = "Mro2Go",
        progress_message = "Generating .go source files.",
        env = {"MROPATH": mropath},
    )
    return [DefaultInfo(files = depset(outs))]

mro2go_codegen = rule(
    attrs = {
        "srcs": attr.label_list(
            allow_empty = False,
            doc = "The `mro_library` targets to use as inputs.",
            providers = [MroInfo],
        ),
        "pipelines": attr.string_list(
            doc = "The list of pipelines for which to generate code. " +
                  "One cannot specify both this and `stages`.",
        ),
        "stages": attr.string_list(
            doc = "The list of stages for which to generate code. " +
                  "One cannot specify both this and `pipelines`.",
        ),
        "package": attr.string(
            doc = "The go package name to use.",
        ),
        "inputs_only": attr.bool(
            doc = "If `True`, do not generate struct types for outputs or chunks.",
        ),
        "_mro2go": attr.label(
            default = "@martian//:mro2go",
            executable = True,
            cfg = "exec",
        ),
    },
    doc = """
Generates a set of `.go` source files based on the given `.mro` inputs, using
`mro2go`.
""",
    implementation = _mro2go_codegen_impl,
)
