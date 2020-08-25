"""Trivial stage to compute z = x^y"""

__MRO__ = """
stage POW(
    in  float x,
    in  float y,
    out float z,
    src py    "stages/pow",
)
"""


def main(args, outs):
    """Computes z = x^y"""
    outs.z = args.x ** args.y
