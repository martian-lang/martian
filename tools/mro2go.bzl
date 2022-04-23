"""Macro for generating go_library targets with `mro2go`."""

load("@bazel_skylib//lib:paths.bzl", "paths")
load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("//tools/private:mro2go.bzl", _mro2go_codegen = "mro2go_codegen")

mro2go_codegen = _mro2go_codegen

def mro2go_library(
        *,
        name,
        srcs,
        importpath,
        main = False,
        pipelines = [],
        stages = [],
        inputs_only = False,
        **kwargs):
    """Creates a go_library target for sources generated using mro2go.

    This will usually be embedded in another go_library target.

    Args:
        name (str): The name of the `go_library` target.
        srcs (list): The `mro_library` targets for which to generate
            go sources.
        importpath (str): The go import path.
        main (bool): If true, use the package name `main`.  Otherwise the
            package name will be the basename of `importpath`.
        pipelines (list, optional): The list of pipelines for which to generate
            code.  One cannot specify both this and `stages`.
        stages (list, optional): The list of stages for which to generate code.
            One cannot specify both this and `pipelines`.
        inputs_only (bool, optional): If `True`, do not generate struct types
            for outputs or chunks. Defaults to False.
        **kwargs: Additional arguments to pass to the `go_library` target, e.g.
            `tags` or `visibility`.
    """
    codegen_name = "_" + name + "_codegen"
    mro2go_codegen(
        name = codegen_name,
        srcs = srcs,
        package = "main" if main else paths.basename(importpath),
        pipelines = pipelines,
        stages = stages,
        inputs_only = inputs_only,
        testonly = kwargs.get("testonly", False),
    )
    go_library(
        name = name,
        srcs = [":" + codegen_name],
        importpath = importpath,
        **kwargs
    )
