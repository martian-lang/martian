"""Defines a rules for making a copy of an executable in a new location."""

def _copy_binary_impl(ctx):
    if not ctx.executable.src:
        fail("binary must be specified", attr = "src")
    dest_file = ctx.actions.declare_file(ctx.attr.dest or ctx.attr.name)
    dest_list = [dest_file]
    if ctx.attr.allow_symlink:
        ctx.actions.symlink(
            output = dest_file,
            target_file = ctx.executable.src,
            is_executable = True,
        )
    else:
        ctx.actions.run(
            executable = "/bin/cp",
            inputs = [ctx.executable.src],
            outputs = dest_list,
            arguments = [
                "--preserve=mode,timestamps",
                "--reflink=auto",
                "-LT",
                ctx.executable.src.path,
                dest_file.path,
            ],
            tools = [],
            mnemonic = "CopyBinary",
            progress_message = "Copying {} to {}.".format(
                ctx.executable.src.short_path,
                dest_file.short_path,
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
        executable = dest_file,
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
        "dest": attr.string(
            doc = "The package-relative location for this file.  " +
                  "This will default to the name of the target.",
        ),
        "allow_symlink": attr.bool(
            default = True,
            doc = "Whether to allow symlinking instead of copying. " +
                  "When False, the output is always a hard copy. " +
                  "When True, the output *can* be a symlink, but there is no " +
                  "guarantee that a symlink is created. Set this to True if " +
                  "you want fast copying and your tools can handle symlinks " +
                  "(which most UNIX tools can), particularly in remote " +
                  "execution environments.",
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

def _dir(p):
    i = p.rfind("/")
    if i == 0:
        return "/"
    if i > 0:
        return p[:i]
    return ""

def _relpath(dest, src):
    dest = _dir(dest)
    if not dest:
        return src
    if src.startswith(dest + "/"):
        return src[len(dest) + 1:]
    parts = dest.split("/")
    prefix = "../"
    for part in parts:
        dest = _dir(dest)
        if src.startswith(dest + "/"):
            return prefix + src[len(dest) + 1:]
        prefix += "../"
    fail(
        "failed to compute a relative path from '{}' to '{}'".format(dest, src),
        "dest",
    )

def _symlink_binary_impl(ctx):
    exe = ctx.attr.src[DefaultInfo].files_to_run.executable
    if not exe:
        if len(ctx.files.src) == 1:
            exe = ctx.files.src[0]
        else:
            fail("binary must be specified", attr = "src")
    dest_file = ctx.actions.declare_file(ctx.attr.dest or ctx.attr.name)
    ctx.actions.run_shell(
        outputs = [dest_file],
        inputs = [exe],
        command = "ln -s \"$1\" \"$2\"",
        arguments = [
            _relpath(
                dest_file.short_path,
                exe.short_path,
            ),
            dest_file.path,
        ],
        mnemonic = "SymlinkExe",
    )
    files = [dest_file] + ctx.files.src
    runfiles = ctx.runfiles(
        files = files,
        transitive_files = depset(transitive = [
            dep[DefaultInfo].files
            for dep in ctx.attr.data
        ]),
    )
    data_runfiles = runfiles.merge(
        ctx.attr.src[DefaultInfo].data_runfiles,
    )
    default_runfiles = runfiles.merge(
        ctx.attr.src[DefaultInfo].default_runfiles,
    )
    for dep in ctx.attr.data:
        info = dep[DefaultInfo]
        data_runfiles = data_runfiles.merge(info.data_runfiles)
        default_runfiles = default_runfiles.merge(info.default_runfiles)
    return [DefaultInfo(
        data_runfiles = data_runfiles,
        default_runfiles = default_runfiles,
        executable = dest_file,
        files = depset(files),
    )]

symlink_binary = rule(
    attrs = {
        "src": attr.label(
            doc = "The executable target to symlink.",
            mandatory = True,
        ),
        "data": attr.label_list(
            doc = "Additional items to add to the runfiles for this target.",
            allow_files = True,
        ),
        "dest": attr.string(
            doc = "The package-relative location for this file.  " +
                  "This will default to the name of the target.",
        ),
    },
    doc = """Creates a symlink to an executable.

Sometimes it is desireable to run a binary from a different name using a
symlink.  This rule creates a symlink to the given `src`, with the same
runfiles.
""",
    executable = True,
    implementation = _symlink_binary_impl,
)
