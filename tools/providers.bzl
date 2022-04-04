"""Definitions of providers used by various rules."""

MroInfo = provider(
    doc = "This rule provides information about required MROPATH",
    fields = {
        "mropath": "Depset of paths to add to MROPATH",
        "transitive_mros": "Depset of mro files in the transitive closure.",
        "stage_py_deps": "Depset of files: Directly depended-on python files, needed in runfiles of `mro_test` rules.",
    },
)
