"""Stage which creates a bunch of files."""

import martian

__MRO__ = """
struct OUTS(
    int  bar,
    file file1,
    txt  file2,
)

stage CREATOR(
    in  int  foo,
    out OUTS bar,
    out txt  file3  "help text"  "output_name.file",
    src py   "creator",
)
"""


def main(args, outs):
    """Create files, some of which are returned in a structure."""
    outs.bar = {
        "bar": args.foo + 3,
        "file1": martian.make_path("file1"),
        "file2": martian.make_path("file2"),
    }
    with open(outs.bar["file1"], "w") as file1:
        file1.write(str(args.foo))
    with open(outs.bar["file2"], "w") as file2:
        file2.write(str(args.foo + 1))
    with open(outs.file3, "w") as file3:
        file3.write(str(args.foo + 2))
