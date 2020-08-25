__MRO__ = """
stage CREATE_DISABLE(
    in  int  foo,
    out bool disable,
    src py   "disable",
)
"""


def main(args, outs):
    outs.disable = args.foo == 1
