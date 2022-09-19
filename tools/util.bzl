"""Shared utility methods for rules."""

def merge_runfiles(
        ctx,
        deps,
        files = [],
        symlinks = {},
        transitive_files = True,
        collect_data = False,
        collect_default = False):
    """Merge runfiles for the given set of dependencies.

    See [ctx.runfiles](https://docs.bazel.build/versions/master/skylark/lib/ctx.html#runfiles)

    Args:
        ctx: The rule execution context.
        deps: Sequence of dependencies, whose runfiles should be in.  If
              `transitive_files` is true, their files will also be included.
        files: Additional files to add to the resulting runfiles.
        symlinks: See [ctx.runfiles](https://docs.bazel.build/versions/master/skylark/lib/ctx.html#runfiles.symlinks)
        transitive_files: If true, include the files from `deps`.
        collect_data: See [ctx.runfiles](https://docs.bazel.build/versions/master/skylark/lib/ctx.html#runfiles.collect_data)
        collect_default: See [ctx.runfiles](https://docs.bazel.build/versions/master/skylark/lib/ctx.html#runfiles.collect_default)

    Returns:
        The merged runfiles object.
    """
    rf = ctx.runfiles(
        files = files,
        transitive_files = depset(transitive = [
            dep[DefaultInfo].files
            for dep in deps
        ]) if transitive_files else None,
        collect_data = collect_data,
        collect_default = collect_default,
        symlinks = symlinks,
    ).merge_all([
        dep[DefaultInfo].default_runfiles
        for dep in deps
    ] + [
        dep[DefaultInfo].data_runfiles
        for dep in deps
    ])
    return rf

def merge_pyinfo(
        deps = [],
        sources = [],
        uses_shared_libraries = False,
        imports = [],
        import_only_deps = [],
        has_py2_only_sources = False,
        has_py3_only_sources = False):
    """Merge transitive PyInfo providers.

    Args:
      deps: sequence of targets with PyInfo providers.
      sources: direct source files to include.
      uses_shared_libraries: The value to use for uses_shared_libraries if
                             it is not True for any dependencies.
      imports: Import paths to add.
      import_only_deps: deps used to compute imports, but nothing else.
      has_py2_only_sources: Whether any of this target's sources require a
                            Python 2 runtime.
      has_py3_only_sources: Whether any of this target's sources require a
                            Python 3 runtime.

    Returns:
        A `PyInfo` provider containing the merged information.
    """
    for dep in deps:
        info = dep[PyInfo]
        if info.uses_shared_libraries:
            uses_shared_libraries = True
        if info.has_py2_only_sources:
            has_py2_only_sources = True
        if info.has_py3_only_sources:
            has_py3_only_sources = True
    return PyInfo(
        has_py2_only_sources = has_py2_only_sources,
        has_py3_only_sources = has_py3_only_sources,
        imports = depset(imports, transitive = [
            dep[PyInfo].imports
            for dep in import_only_deps + deps
        ]),
        transitive_sources = depset(sources, transitive = [
            dep[PyInfo].transitive_sources
            for dep in deps
        ]),
        uses_shared_libraries = uses_shared_libraries,
    )
