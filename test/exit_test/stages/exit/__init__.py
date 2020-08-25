import martian

__MRO__ = """
stage EXIT(
    in  string message,
    out string empty,
    src py     "stages/exit",
)
"""


def main(args, outs):
    outs.empty = ""
    martian.exit(b"Goodbye \xc2\xc2 World!")
