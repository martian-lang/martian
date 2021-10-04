"""A bazel rule to copy a single output file from a target to a new location."""

load("@bazel_skylib//lib:paths.bzl", "paths")

def _extract_file_impl(ctx):
    if not ctx.files.srcs:
        fail("Source has no output files.", attr = "srcs")
    match = ctx.attr.match or paths.basename(ctx.attr.dest)
    files = [f for f in ctx.files.srcs if f.short_path.endswith(match)]
    if not files:
        fail(
            "No output files have suffix",
            match,
            "\nFiles:\n\t",
            "\n\t".join([f.short_path for f in ctx.files.src]),
            attr = "match",
        )
    if len(files) > 1:
        fail(
            "Multiple files match suffix:\n\t",
            "\n\t".join([f.short_path for f in files]),
        )
    dest_file = ctx.actions.declare_file(ctx.attr.dest or ctx.attr.name)
    dest_list = [dest_file]
    ctx.actions.run(
        executable = "/bin/cp",
        inputs = files,
        outputs = dest_list,
        arguments = [
            "--preserve=mode,timestamps",
            "--reflink=auto",
            "-LT",
            files[0].path,
            dest_file.path,
        ],
        tools = [],
        mnemonic = "ExtractFile",
        progress_message = "Copying {} to {}.".format(
            files[0].short_path,
            dest_file.short_path,
        ),
    )
    return [DefaultInfo(
        files = depset(dest_list),
    )]

extract_file = rule(
    attrs = {
        "srcs": attr.label_list(
            doc = "The target generating the file to copy.",
            mandatory = True,
            cfg = "target",
        ),
        "match": attr.string(
            doc = "The suffix on the path for the desired file.  " +
                  "Must match exactly one file from `src`.  " +
                  "If `dest` is set, defaults to the base name of `dest`.",
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
    doc = """Copies a file to new location.

Take the single file with the given suffix the given build target, and copy it
to the given destination location.
""",
    implementation = _extract_file_impl,
)
