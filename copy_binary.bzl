def _copy_binary_impl(ctx):
    if not ctx.executable.src:
        fail("binary must be specified", attr = "src")
    basename = ctx.executable.src.basename
    dest_list = [ctx.outputs.dest]
    ctx.actions.run(
        executable = "install",
        inputs = [ctx.executable.src],
        outputs = dest_list,
        arguments = [
            "-DTp",
            ctx.executable.src.path,
            ctx.outputs.dest.path,
        ],
        mnemonic = "CopyBinary",
        progress_message = "Copying {} to {}.".format(
            ctx.executable.src.short_path,
            ctx.outputs.dest.short_path,
        ),
    )
    data_runfiles = ctx.attr.src.data_runfiles.files.to_list()
    data_runfiles.remove(ctx.executable.src)
    default_runfiles = ctx.attr.src.default_runfiles.files.to_list()
    default_runfiles.remove(ctx.executable.src)
    files = dest_list + ctx.files.data
    data_runfiles = ctx.runfiles(
        files = files,
        transitive_files = depset(data_runfiles),
        collect_data = False,
        collect_default = False,
    )
    default_runfiles = ctx.runfiles(
        files = files,
        transitive_files = depset(default_runfiles),
        collect_data = False,
        collect_default = False,
    )
    for dep in ctx.attr.data:
        info = dep[DefaultInfo]
        data_runfiles = data_runfiles.merge(info.data_runfiles)
        default_runfiles = default_runfiles.merge(info.default_runfiles)
    return [DefaultInfo(
        data_runfiles = data_runfiles,
        default_runfiles = default_runfiles,
        executable = ctx.outputs.dest,
        files = depset(dest_list),
    )]

copy_binary = rule(
    attrs = {
        "src": attr.label(
            doc = "The executable target to copy.",
            executable = True,
            mandatory = True,
            cfg = "target",
        ),
        "data": attr.label_list(
            doc = "Additional items to add to the runfiles for this target.",
            allow_files = True,
        ),
        "dest": attr.output(
            doc = "The package-relative location for this file.",
            mandatory = True,
        ),
    },
    doc = """Copies an executable to new location.

Bazel places binaries in the directory of the package which declares the
corresponding target, but this is not always the desired behavior.  This
rule allows a user to place the binary in a new location.
""",
    executable = True,
    implementation = _copy_binary_impl,
)
